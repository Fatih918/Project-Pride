package quarantine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"pride-plugin-module/installation"
	"pride-plugin-module/storage"
)

const (
	DefaultMaxAge       = 30 * 24 * time.Hour
	DefaultMaxArtifacts = 100
	DefaultMaxBytes     = int64(1 << 30)
)

type Policy struct {
	MaxAge       time.Duration
	MaxArtifacts int
	MaxBytes     int64
}

func DefaultPolicy() Policy {
	return Policy{MaxAge: DefaultMaxAge, MaxArtifacts: DefaultMaxArtifacts, MaxBytes: DefaultMaxBytes}
}

func (p Policy) Validate() error {
	if p.MaxAge <= 0 || p.MaxArtifacts <= 0 || p.MaxBytes <= 0 {
		return errors.New("quarantine retention limits must be positive")
	}
	return nil
}

type OperationLookup interface {
	GetOperation(context.Context, installation.OperationID) (installation.InstallationOperation, error)
}

type Storage interface {
	ListQuarantineEntries(context.Context) ([]storage.QuarantineEntry, error)
	DeleteQuarantineOperation(context.Context, string) error
}

type Report struct {
	Scanned       int
	Removed       []string
	Retained      int
	RetainedBytes int64
}

type Retainer struct {
	operations OperationLookup
	storage    Storage
	policy     Policy
	now        func() time.Time
}

func New(operations OperationLookup, quarantineStorage Storage, policy Policy) (*Retainer, error) {
	if operations == nil || quarantineStorage == nil {
		return nil, errors.New("quarantine retention dependencies are required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Retainer{
		operations: operations,
		storage:    quarantineStorage,
		policy:     policy,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

// Prune removes only quarantine directories whose matching journal operation
// is terminal (FAILED or ROLLED_BACK). Unknown and non-terminal operations are
// retained. Age limits are applied first; count and byte limits then evict the
// oldest eligible entries until the remaining inventory is within policy.
func (r *Retainer) Prune(ctx context.Context) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	entries, err := r.storage.ListQuarantineEntries(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("inventory Plugin quarantine: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ModifiedAt.Equal(entries[j].ModifiedAt) {
			return entries[i].OperationID < entries[j].OperationID
		}
		return entries[i].ModifiedAt.Before(entries[j].ModifiedAt)
	})

	type candidate struct {
		entry    storage.QuarantineEntry
		eligible bool
	}
	items := make([]candidate, 0, len(entries))
	// Resolve every journal record before deleting anything. A persistence
	// failure therefore cannot leave a partially pruned inventory.
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		op, lookupErr := r.operations.GetOperation(ctx, installation.OperationID(entry.OperationID))
		if errors.Is(lookupErr, installation.ErrOperationNotFound) {
			items = append(items, candidate{entry: entry})
			continue
		}
		if lookupErr != nil {
			return Report{}, fmt.Errorf("resolve quarantine operation %q: %w", entry.OperationID, lookupErr)
		}
		eligible := op.OperationID == installation.OperationID(entry.OperationID) &&
			(op.Phase == installation.PhaseFailed || op.Phase == installation.PhaseRolledBack)
		items = append(items, candidate{entry: entry, eligible: eligible})
	}

	now := r.now().UTC()
	report := Report{Scanned: len(items)}
	count := len(items)
	var bytes int64
	for _, item := range items {
		if item.entry.SizeBytes < 0 || item.entry.SizeBytes > int64(^uint64(0)>>1)-bytes {
			return Report{}, errors.New("quarantine inventory size overflow")
		}
		bytes += item.entry.SizeBytes
	}

	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		overLimit := count > r.policy.MaxArtifacts || bytes > r.policy.MaxBytes
		expired := now.Sub(item.entry.ModifiedAt) >= r.policy.MaxAge
		if !item.eligible || (!expired && !overLimit) {
			continue
		}
		if err := r.storage.DeleteQuarantineOperation(ctx, item.entry.OperationID); err != nil {
			return report, fmt.Errorf("delete expired/over-limit quarantine operation %q: %w", item.entry.OperationID, err)
		}
		report.Removed = append(report.Removed, item.entry.OperationID)
		count--
		bytes -= item.entry.SizeBytes
	}
	report.Retained = count
	report.RetainedBytes = bytes
	return report, nil
}
