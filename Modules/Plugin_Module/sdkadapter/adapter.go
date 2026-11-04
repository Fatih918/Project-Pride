// Package sdkadapter translates Plugin Module discovery operations into the
// SDK-neutral port consumed by the PRIDE Plugin SDK server adapter.
package sdkadapter

import (
	"context"
	"errors"
	"fmt"

	"pride-plugin-module/discovery"
	"pride-plugin-module/domain"
	"pride-plugin-module/registry"
	"pride-plugin-sdk/server"
)

// Adapter implements server.DiscoveryPort using the Plugin Module discovery
// application service. It contains transport-boundary mapping only; business
// rules and registry semantics remain in the Plugin Module.
type Adapter struct {
	service *discovery.Service
}

func New(service *discovery.Service) (*Adapter, error) {
	if service == nil {
		return nil, errors.New("plugin module sdk adapter: discovery service is required")
	}
	return &Adapter{service: service}, nil
}

func (a *Adapter) ListPlugins(ctx context.Context, query server.ListQuery) (server.PluginPage, error) {
	pluginType, err := toDomainType(query.Type)
	if err != nil {
		return server.PluginPage{}, mapError(err)
	}
	pluginState, err := toDomainState(query.State)
	if err != nil {
		return server.PluginPage{}, mapError(err)
	}

	page, err := a.service.ListPlugins(ctx, discovery.ListQuery{
		PageSize: query.PageSize,
		PageToken: query.PageToken,
		Type: pluginType,
		State: pluginState,
		Metadata: query.Metadata,
	})
	if err != nil {
		return server.PluginPage{}, mapError(err)
	}

	plugins, err := mapPluginRecords(page.Plugins)
	if err != nil {
		return server.PluginPage{}, mapError(err)
	}
	return server.PluginPage{
		Plugins: plugins,
		NextPageToken: page.NextPageToken,
		ResponseMetadata: page.ResponseMetadata,
	}, nil
}

func (a *Adapter) GetPlugin(ctx context.Context, query server.PluginQuery) (server.Plugin, error) {
	record, err := a.service.GetPlugin(ctx, discovery.PluginQuery{
		PluginID: query.PluginID,
		Metadata: query.Metadata,
	})
	if err != nil {
		return server.Plugin{}, mapError(err)
	}
	plugin, err := mapPluginRecord(record)
	if err != nil {
		return server.Plugin{}, mapError(err)
	}
	return plugin, nil
}

func (a *Adapter) FindPluginsByCapability(ctx context.Context, query server.CapabilityQuery) (server.PluginPage, error) {
	page, err := a.service.FindPluginsByCapability(ctx, discovery.CapabilityQuery{
		CapabilityID: query.CapabilityID,
		PageSize: query.PageSize,
		PageToken: query.PageToken,
		Metadata: query.Metadata,
	})
	if err != nil {
		return server.PluginPage{}, mapError(err)
	}

	plugins, err := mapPluginRecords(page.Plugins)
	if err != nil {
		return server.PluginPage{}, mapError(err)
	}
	return server.PluginPage{
		Plugins: plugins,
		NextPageToken: page.NextPageToken,
		ResponseMetadata: page.ResponseMetadata,
	}, nil
}

func (a *Adapter) GetPluginMetadata(ctx context.Context, query server.PluginQuery) (server.PluginMetadata, error) {
	value, err := a.service.GetPluginMetadata(ctx, discovery.PluginQuery{
		PluginID: query.PluginID,
		Metadata: query.Metadata,
	})
	if err != nil {
		return server.PluginMetadata{}, mapError(err)
	}

	pluginType, err := fromDomainType(value.Type)
	if err != nil {
		return server.PluginMetadata{}, mapError(err)
	}
	return server.PluginMetadata{
		PluginID: value.PluginID,
		Name: value.Name,
		Version: value.Version,
		Type: pluginType,
		Capabilities: mapCapabilities(value.Capabilities),
		PermissionIDs: append([]string(nil), value.PermissionIDs...),
		CompatibilityStatus: value.CompatibilityStatus,
		Attributes: value.Attributes,
		ResponseMetadata: value.ResponseMetadata,
	}, nil
}

func (a *Adapter) GetPluginState(ctx context.Context, query server.PluginQuery) (server.PluginStateInfo, error) {
	value, err := a.service.GetPluginState(ctx, discovery.PluginQuery{
		PluginID: query.PluginID,
		Metadata: query.Metadata,
	})
	if err != nil {
		return server.PluginStateInfo{}, mapError(err)
	}

	state, err := fromDomainState(value.State)
	if err != nil {
		return server.PluginStateInfo{}, mapError(err)
	}
	return server.PluginStateInfo{
		PluginID: value.PluginID,
		State: state,
		Available: value.Available,
		StatusMessage: value.StatusMessage,
		ResponseMetadata: value.ResponseMetadata,
	}, nil
}

func mapPluginRecords(values []discovery.PluginRecord) ([]server.Plugin, error) {
	plugins := make([]server.Plugin, 0, len(values))
	for _, value := range values {
		plugin, err := mapPluginRecord(value)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}

func mapPluginRecord(value discovery.PluginRecord) (server.Plugin, error) {
	pluginType, err := fromDomainType(value.Type)
	if err != nil {
		return server.Plugin{}, err
	}
	state, err := fromDomainState(value.State)
	if err != nil {
		return server.Plugin{}, err
	}
	return server.Plugin{
		Manifest: server.Manifest{
			PluginID: string(value.Manifest.PluginID),
			Name: value.Manifest.Name,
			Version: value.Manifest.Version,
			Description: value.Manifest.Description,
			Author: value.Manifest.Author,
			License: value.Manifest.License,
			EntryPoint: value.Manifest.EntryPoint,
			MinimumPRIDEVersion: value.Manifest.MinimumPRIDEVersion,
			Homepage: value.Manifest.Homepage,
			DocumentationURL: value.Manifest.DocumentationURL,
			RepositoryURL: value.Manifest.RepositoryURL,
			IssueTrackerURL: value.Manifest.IssueTrackerURL,
			Keywords: append([]string(nil), value.Manifest.Keywords...),
			Category: value.Manifest.Category,
			Icon: value.Manifest.Icon,
			ChangelogURL: value.Manifest.ChangelogURL,
		},
		Type: pluginType,
		State: state,
		Capabilities: mapCapabilities(value.Capabilities),
		Available: value.Available,
		CompatibilityStatus: value.CompatibilityStatus,
	}, nil
}

func mapCapabilities(values []domain.Capability) []server.Capability {
	capabilities := make([]server.Capability, 0, len(values))
	for _, value := range values {
		capabilities = append(capabilities, server.Capability{
			ID: value.ID,
			Name: value.Name,
			Description: value.Description,
		})
	}
	return capabilities
}

func toDomainType(value server.PluginType) (domain.PluginType, error) {
	switch value {
	case server.PluginTypeUnspecified:
		return domain.PluginTypeUnspecified, nil
	case server.PluginTypeActive:
		return domain.PluginTypeActive, nil
	case server.PluginTypePassive:
		return domain.PluginTypePassive, nil
	default:
		return domain.PluginTypeUnspecified, fmt.Errorf("%w: unknown Plugin type filter", discovery.ErrInvalidQuery)
	}
}

func fromDomainType(value domain.PluginType) (server.PluginType, error) {
	switch value {
	case domain.PluginTypeUnspecified:
		return server.PluginTypeUnspecified, nil
	case domain.PluginTypeActive:
		return server.PluginTypeActive, nil
	case domain.PluginTypePassive:
		return server.PluginTypePassive, nil
	default:
		return server.PluginTypeUnspecified, fmt.Errorf("unknown Plugin type returned by Module")
	}
}

func toDomainState(value server.PluginState) (domain.PluginState, error) {
	switch value {
	case server.PluginStateUnspecified:
		return domain.PluginStateUnspecified, nil
	case server.PluginStateInstalled:
		return domain.PluginStateInstalled, nil
	case server.PluginStateEnabled:
		return domain.PluginStateEnabled, nil
	case server.PluginStateDisabled:
		return domain.PluginStateDisabled, nil
	case server.PluginStateRunning:
		return domain.PluginStateRunning, nil
	case server.PluginStateError:
		return domain.PluginStateError, nil
	default:
		return domain.PluginStateUnspecified, fmt.Errorf("%w: unknown Plugin state filter", discovery.ErrInvalidQuery)
	}
}

func fromDomainState(value domain.PluginState) (server.PluginState, error) {
	switch value {
	case domain.PluginStateUnspecified:
		return server.PluginStateUnspecified, nil
	case domain.PluginStateInstalled:
		return server.PluginStateInstalled, nil
	case domain.PluginStateEnabled:
		return server.PluginStateEnabled, nil
	case domain.PluginStateDisabled:
		return server.PluginStateDisabled, nil
	case domain.PluginStateRunning:
		return server.PluginStateRunning, nil
	case domain.PluginStateError:
		return server.PluginStateError, nil
	default:
		return server.PluginStateUnspecified, errors.New("unknown Plugin state returned by Module")
	}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	switch {
	case errors.Is(err, discovery.ErrInvalidQuery):
		return &server.DomainError{
			Code: server.CodeInvalidInput,
			Message: "Plugin discovery query is invalid",
			Cause: err,
		}
	case errors.Is(err, discovery.ErrPluginNotFound), errors.Is(err, registry.ErrNotFound):
		return &server.DomainError{
			Code: server.CodePluginNotFound,
			Message: "Plugin was not found",
			Cause: err,
		}
	default:
		return &server.DomainError{
			Code: server.CodeInternal,
			Message: "Plugin discovery failed",
			Cause: err,
		}
	}
}

var _ server.DiscoveryPort = (*Adapter)(nil)
