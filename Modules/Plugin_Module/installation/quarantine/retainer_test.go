package quarantine

import (
	"context"
	"errors"
	"testing"
	"time"

	"pride-plugin-module/installation"
	"pride-plugin-module/storage"
)

type fakeOperations struct {
	operations map[installation.OperationID]installation.InstallationOperation
	err        error
}

func (f *fakeOperations) GetOperation(_ context.Context, id installation.OperationID) (installation.InstallationOperation, error) {
	if f.err != nil {
		return installation.InstallationOperation{}, f.err
	}
	op, ok := f.operations[id]
	if !ok {
		return installation.InstallationOperation{}, installation.ErrOperationNotFound
	}
	return op, nil
}

type fakeStorage struct {
	entries   []storage.QuarantineEntry
	removed   []string
	listErr   error
	deleteErr error
}

func (f *fakeStorage) ListQuarantineEntries(context.Context) ([]storage.QuarantineEntry, error) {
	return append([]storage.QuarantineEntry(nil), f.entries...), f.listErr
}

func (f *fakeStorage) DeleteQuarantineOperation(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.removed = append(f.removed, id)
	return nil
}

func TestPruneRemovesExpiredTerminalOnly(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	operations := &fakeOperations{operations: map[installation.OperationID]installation.InstallationOperation{
		"old-failed": {OperationID: "old-failed", Phase: installation.PhaseFailed},
		"old-active": {OperationID: "old-active", Phase: installation.PhasePromoted},
	}}
	store := &fakeStorage{entries: []storage.QuarantineEntry{
		{OperationID: "old-failed", ModifiedAt: now.Add(-31 * 24 * time.Hour), SizeBytes: 100},
		{OperationID: "old-active", ModifiedAt: now.Add(-40 * 24 * time.Hour), SizeBytes: 200},
		{OperationID: "missing-journal", ModifiedAt: now.Add(-50 * 24 * time.Hour), SizeBytes: 300},
	}}
	retainer, err := New(operations, store, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	retainer.now = func() time.Time { return now }

	report, err := retainer.Prune(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Removed) != 1 || report.Removed[0] != "old-failed" {
		t.Fatalf("removed = %v", report.Removed)
	}
	if report.Retained != 2 || report.RetainedBytes != 500 {
		t.Fatalf("unexpected retained inventory: %+v", report)
	}
}

func TestPruneEvictsOldestEligibleUntilCountAndBytesFit(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	ops := &fakeOperations{operations: map[installation.OperationID]installation.InstallationOperation{
		"active":   {OperationID: "active", Phase: installation.PhaseStaged},
		"failed-a": {OperationID: "failed-a", Phase: installation.PhaseFailed},
		"failed-b": {OperationID: "failed-b", Phase: installation.PhaseRolledBack},
	}}
	store := &fakeStorage{entries: []storage.QuarantineEntry{
		{OperationID: "active", ModifiedAt: now.Add(-3 * time.Hour), SizeBytes: 60},
		{OperationID: "failed-a", ModifiedAt: now.Add(-2 * time.Hour), SizeBytes: 60},
		{OperationID: "failed-b", ModifiedAt: now.Add(-time.Hour), SizeBytes: 60},
	}}
	policy := Policy{MaxAge: 24 * time.Hour, MaxArtifacts: 2, MaxBytes: 100}
	retainer, err := New(ops, store, policy)
	if err != nil {
		t.Fatal(err)
	}
	retainer.now = func() time.Time { return now }

	report, err := retainer.Prune(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Removed) != 2 || report.Removed[0] != "failed-a" || report.Removed[1] != "failed-b" {
		t.Fatalf("removed = %v", report.Removed)
	}
	if report.Retained != 1 || report.RetainedBytes != 60 {
		t.Fatalf("unexpected retained inventory: %+v", report)
	}
}

func TestPruneFailsBeforeDeletionWhenJournalLookupFails(t *testing.T) {
	store := &fakeStorage{entries: []storage.QuarantineEntry{{OperationID: "failed", SizeBytes: 1}}}
	ops := &fakeOperations{err: errors.New("database unavailable")}
	retainer, err := New(ops, store, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := retainer.Prune(context.Background()); err == nil {
		t.Fatal("expected journal lookup failure")
	}
	if len(store.removed) != 0 {
		t.Fatalf("deleted before journal resolution: %v", store.removed)
	}
}

func TestNewRejectsInvalidPolicy(t *testing.T) {
	for _, policy := range []Policy{
		{},
		{MaxAge: time.Hour, MaxArtifacts: 0, MaxBytes: 1},
		{MaxAge: time.Hour, MaxArtifacts: 1, MaxBytes: -1},
	} {
		if _, err := New(&fakeOperations{}, &fakeStorage{}, policy); err == nil {
			t.Errorf("expected invalid policy rejection: %+v", policy)
		}
	}
}
