package discovery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"pride-plugin-module/domain"
	"pride-plugin-module/registry"
)

// Service exposes read-only queries over the authoritative installed Plugin
// Registry. It does not perform capability ranking, authorization, lifecycle,
// package loading, or execution.
type Service struct {
	repository registry.Repository
}

func NewService(repository registry.Repository) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: registry repository is required", ErrInvalidQuery)
	}
	return &Service{repository: repository}, nil
}

func (s *Service) ListPlugins(ctx context.Context, query ListQuery) (PluginPage, error) {
	pageSize, err := normalizePageSize(query.PageSize)
	if err != nil {
		return PluginPage{}, err
	}
	if !validTypeFilter(query.Type) || !validStateFilter(query.State) {
		return PluginPage{}, fmt.Errorf("%w: unknown type or state filter", ErrInvalidQuery)
	}

	plugins, err := s.repository.List(ctx)
	if err != nil {
		return PluginPage{}, err
	}
	filtered := make([]domain.Plugin, 0, len(plugins))
	for _, plugin := range plugins {
		if query.Type != domain.PluginTypeUnspecified && plugin.Type != query.Type {
			continue
		}
		if query.State != domain.PluginStateUnspecified && plugin.State != query.State {
			continue
		}
		filtered = append(filtered, plugin)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })
	return buildPage(filtered, pageSize, query.PageToken, fmt.Sprintf("list:%d:%d", query.Type, query.State))
}

func (s *Service) GetPlugin(ctx context.Context, query PluginQuery) (PluginRecord, error) {
	plugin, err := s.getPlugin(ctx, query.PluginID)
	if err != nil {
		return PluginRecord{}, err
	}
	return toPluginRecord(plugin), nil
}

func (s *Service) FindPluginsByCapability(ctx context.Context, query CapabilityQuery) (PluginPage, error) {
	if strings.TrimSpace(query.CapabilityID) == "" {
		return PluginPage{}, fmt.Errorf("%w: capability ID is required", ErrInvalidQuery)
	}
	pageSize, err := normalizePageSize(query.PageSize)
	if err != nil {
		return PluginPage{}, err
	}

	plugins, err := s.repository.List(ctx)
	if err != nil {
		return PluginPage{}, err
	}
	matches := make([]domain.Plugin, 0)
	for _, plugin := range plugins {
		for _, capability := range plugin.Protected.Capabilities {
			if capability.ID == query.CapabilityID {
				matches = append(matches, plugin)
				break
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].ID < matches[j].ID })
	return buildPage(matches, pageSize, query.PageToken, "capability:"+query.CapabilityID)
}

func (s *Service) GetPluginMetadata(ctx context.Context, query PluginQuery) (PluginMetadata, error) {
	plugin, err := s.getPlugin(ctx, query.PluginID)
	if err != nil {
		return PluginMetadata{}, err
	}
	plugin = plugin.Clone()
	return PluginMetadata{
		PluginID:            string(plugin.ID),
		Name:                plugin.Manifest.Name,
		Version:             plugin.Manifest.Version,
		Type:                plugin.Type,
		Capabilities:        plugin.Protected.Capabilities,
		PermissionIDs:       plugin.Protected.PermissionIDs,
		CompatibilityStatus: plugin.Protected.CompatibilityStatus,
		Attributes:          plugin.Protected.Attributes,
	}, nil
}

func (s *Service) GetPluginState(ctx context.Context, query PluginQuery) (PluginStateInfo, error) {
	plugin, err := s.getPlugin(ctx, query.PluginID)
	if err != nil {
		return PluginStateInfo{}, err
	}
	return PluginStateInfo{
		PluginID:      string(plugin.ID),
		State:         plugin.State,
		Available:     plugin.Available,
		StatusMessage: plugin.StatusMessage,
	}, nil
}

func (s *Service) getPlugin(ctx context.Context, id string) (domain.Plugin, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Plugin{}, fmt.Errorf("%w: plugin ID is required", ErrInvalidQuery)
	}
	plugin, err := s.repository.Get(ctx, domain.PluginID(id))
	if err != nil {
		if errors.Is(err, registry.ErrNotFound) {
			return domain.Plugin{}, fmt.Errorf("%w: %s", ErrPluginNotFound, id)
		}
		return domain.Plugin{}, err
	}
	return plugin, nil
}

func toPluginRecord(plugin domain.Plugin) PluginRecord {
	plugin = plugin.Clone()
	return PluginRecord{
		Manifest:            plugin.Manifest,
		Type:                plugin.Type,
		State:               plugin.State,
		Capabilities:        plugin.Protected.Capabilities,
		Available:           plugin.Available,
		CompatibilityStatus: plugin.Protected.CompatibilityStatus,
	}
}

func buildPage(plugins []domain.Plugin, pageSize int, token, filter string) (PluginPage, error) {
	digest, err := snapshotDigest(plugins, filter)
	if err != nil {
		return PluginPage{}, err
	}
	offset, err := decodePageToken(token, digest, len(plugins))
	if err != nil {
		return PluginPage{}, err
	}
	page, next := paginate(plugins, pageSize, offset, digest)
	records := make([]PluginRecord, 0, len(page))
	for _, plugin := range page {
		records = append(records, toPluginRecord(plugin))
	}
	return PluginPage{Plugins: records, NextPageToken: next}, nil
}

func validTypeFilter(value domain.PluginType) bool {
	return value == domain.PluginTypeUnspecified ||
		value == domain.PluginTypeActive ||
		value == domain.PluginTypePassive
}

func validStateFilter(value domain.PluginState) bool {
	return value == domain.PluginStateUnspecified ||
		(value >= domain.PluginStateInstalled && value <= domain.PluginStateError)
}
