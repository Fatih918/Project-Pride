package discovery

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"pride-plugin-module/domain"
	"pride-plugin-module/registry"
)

func newTestService(t *testing.T, plugins ...domain.Plugin) *Service {
	t.Helper()
	repo := registry.NewMemoryRepository()
	for _, plugin := range plugins {
		if err := repo.Create(context.Background(), plugin); err != nil {
			t.Fatalf("Create(%q) error = %v", plugin.ID, err)
		}
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func discoveryPlugin(id string, pluginType domain.PluginType, state domain.PluginState, capability string) domain.Plugin {
	pluginID := domain.PluginID(id)
	capabilities := []domain.Capability(nil)
	if capability != "" {
		capabilities = []domain.Capability{{ID: capability, Name: capability + " capability"}}
	}
	return domain.Plugin{
		ID: pluginID,
		Manifest: domain.Manifest{
			PluginID:            pluginID,
			Name:                id,
			Version:             "1.0.0",
			Description:         "Discovery test plugin",
			Author:              "PRIDE tests",
			License:             "MIT",
			EntryPoint:          "main.go",
			MinimumPRIDEVersion: "1.0.0",
			Keywords:            []string{"test"},
		},
		Type:  pluginType,
		State: state,
		Protected: domain.ProtectedMetadata{
			Capabilities:        capabilities,
			PermissionIDs:       []string{"notes.read"},
			CompatibilityStatus: "compatible",
			Attributes:          map[string]any{"source": "test"},
		},
		Available: true,
	}
}

func TestListPluginsReturnsEmptyPageForEmptyRegistry(t *testing.T) {
	service := newTestService(t)
	page, err := service.ListPlugins(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("ListPlugins() error = %v", err)
	}
	if len(page.Plugins) != 0 || page.NextPageToken != "" {
		t.Fatalf("ListPlugins() = %+v, want empty page", page)
	}
}

func TestListPluginsAppliesTypeAndStateFilters(t *testing.T) {
	service := newTestService(t,
		discoveryPlugin("com.example.alpha", domain.PluginTypeActive, domain.PluginStateEnabled, ""),
		discoveryPlugin("com.example.beta", domain.PluginTypePassive, domain.PluginStateInstalled, ""),
		discoveryPlugin("com.example.gamma", domain.PluginTypeActive, domain.PluginStateDisabled, ""),
	)
	page, err := service.ListPlugins(context.Background(), ListQuery{
		Type: domain.PluginTypeActive, State: domain.PluginStateEnabled,
	})
	if err != nil {
		t.Fatalf("ListPlugins() error = %v", err)
	}
	if len(page.Plugins) != 1 || page.Plugins[0].Manifest.PluginID != "com.example.alpha" {
		t.Fatalf("ListPlugins() = %+v, want only active/enabled alpha", page.Plugins)
	}
}

func TestListPluginsPaginatesDeterministically(t *testing.T) {
	service := newTestService(t,
		discoveryPlugin("com.example.charlie", domain.PluginTypeActive, domain.PluginStateInstalled, ""),
		discoveryPlugin("com.example.alpha", domain.PluginTypeActive, domain.PluginStateInstalled, ""),
		discoveryPlugin("com.example.bravo", domain.PluginTypeActive, domain.PluginStateInstalled, ""),
	)
	first, err := service.ListPlugins(context.Background(), ListQuery{PageSize: 2})
	if err != nil {
		t.Fatalf("first ListPlugins() error = %v", err)
	}
	if len(first.Plugins) != 2 || first.NextPageToken == "" {
		t.Fatalf("first page = %+v, want two records and a next token", first)
	}
	if first.Plugins[0].Manifest.PluginID != "com.example.alpha" ||
		first.Plugins[1].Manifest.PluginID != "com.example.bravo" {
		t.Fatalf("first page order = %q, %q", first.Plugins[0].Manifest.PluginID, first.Plugins[1].Manifest.PluginID)
	}

	second, err := service.ListPlugins(context.Background(), ListQuery{PageSize: 2, PageToken: first.NextPageToken})
	if err != nil {
		t.Fatalf("second ListPlugins() error = %v", err)
	}
	if len(second.Plugins) != 1 || second.NextPageToken != "" ||
		second.Plugins[0].Manifest.PluginID != "com.example.charlie" {
		t.Fatalf("second page = %+v, want only charlie and no next token", second)
	}
}

func TestListPluginsRejectsMalformedAndStaleTokens(t *testing.T) {
	service := newTestService(t,
		discoveryPlugin("com.example.alpha", domain.PluginTypeActive, domain.PluginStateInstalled, ""),
		discoveryPlugin("com.example.beta", domain.PluginTypeActive, domain.PluginStateInstalled, ""),
	)
	first, err := service.ListPlugins(context.Background(), ListQuery{PageSize: 1})
	if err != nil {
		t.Fatalf("ListPlugins() error = %v", err)
	}
	if _, err := service.ListPlugins(context.Background(), ListQuery{PageSize: 1, PageToken: "not-a-token"}); !errors.Is(err, ErrInvalidPageToken) {
		t.Fatalf("malformed token error = %v, want ErrInvalidPageToken", err)
	}
	if err := service.repository.(*registry.MemoryRepository).Create(context.Background(),
		discoveryPlugin("com.example.gamma", domain.PluginTypeActive, domain.PluginStateInstalled, "")); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.ListPlugins(context.Background(), ListQuery{PageSize: 1, PageToken: first.NextPageToken}); !errors.Is(err, ErrInvalidPageToken) {
		t.Fatalf("stale token error = %v, want ErrInvalidPageToken", err)
	}
}

func TestFindPluginsByCapabilityUsesExactIdentifier(t *testing.T) {
	service := newTestService(t,
		discoveryPlugin("com.example.alpha", domain.PluginTypeActive, domain.PluginStateInstalled, "notes.read"),
		discoveryPlugin("com.example.beta", domain.PluginTypePassive, domain.PluginStateInstalled, "notes.read"),
		discoveryPlugin("com.example.gamma", domain.PluginTypeActive, domain.PluginStateInstalled, "notes.write"),
	)
	page, err := service.FindPluginsByCapability(context.Background(), CapabilityQuery{CapabilityID: "notes.read"})
	if err != nil {
		t.Fatalf("FindPluginsByCapability() error = %v", err)
	}
	if len(page.Plugins) != 2 {
		t.Fatalf("FindPluginsByCapability() returned %d plugins, want 2", len(page.Plugins))
	}
	empty, err := service.FindPluginsByCapability(context.Background(), CapabilityQuery{CapabilityID: "notes.unknown"})
	if err != nil {
		t.Fatalf("FindPluginsByCapability() unknown capability error = %v", err)
	}
	if len(empty.Plugins) != 0 {
		t.Fatalf("unknown capability returned %d plugins, want 0", len(empty.Plugins))
	}
}

func TestGetPluginReturnsNotFound(t *testing.T) {
	service := newTestService(t)
	_, err := service.GetPlugin(context.Background(), PluginQuery{PluginID: "missing"})
	if !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("GetPlugin() error = %v, want ErrPluginNotFound", err)
	}
}

func TestGetPluginMetadataReturnsProtectedMetadata(t *testing.T) {
	service := newTestService(t, discoveryPlugin("com.example.notes", domain.PluginTypeActive, domain.PluginStateEnabled, "notes.read"))
	metadata, err := service.GetPluginMetadata(context.Background(), PluginQuery{PluginID: "com.example.notes"})
	if err != nil {
		t.Fatalf("GetPluginMetadata() error = %v", err)
	}
	if metadata.PluginID != "com.example.notes" || len(metadata.Capabilities) != 1 ||
		len(metadata.PermissionIDs) != 1 || metadata.CompatibilityStatus != "compatible" {
		t.Fatalf("GetPluginMetadata() = %+v", metadata)
	}
	metadata.Attributes["source"] = "mutated"
	again, err := service.GetPluginMetadata(context.Background(), PluginQuery{PluginID: "com.example.notes"})
	if err != nil {
		t.Fatalf("second GetPluginMetadata() error = %v", err)
	}
	if again.Attributes["source"] != "test" {
		t.Fatal("metadata response shares mutable attributes with registry")
	}
}

func TestGetPluginStateReturnsStoredOperationalState(t *testing.T) {
	plugin := discoveryPlugin("com.example.notes", domain.PluginTypePassive, domain.PluginStateDisabled, "")
	plugin.Available = false
	plugin.StatusMessage = "Disabled by administrator"
	service := newTestService(t, plugin)
	state, err := service.GetPluginState(context.Background(), PluginQuery{PluginID: "com.example.notes"})
	if err != nil {
		t.Fatalf("GetPluginState() error = %v", err)
	}
	if state.State != domain.PluginStateDisabled || state.Available || state.StatusMessage != "Disabled by administrator" {
		t.Fatalf("GetPluginState() = %+v", state)
	}
}

func TestDiscoveryRejectsInvalidQueries(t *testing.T) {
	service := newTestService(t)
	cases := []struct {
		name string
		run  func() error
	}{
		{"negative page size", func() error {
			_, err := service.ListPlugins(context.Background(), ListQuery{PageSize: -1})
			return err
		}},
		{"unknown type", func() error {
			_, err := service.ListPlugins(context.Background(), ListQuery{Type: domain.PluginType(99)})
			return err
		}},
		{"empty plugin ID", func() error {
			_, err := service.GetPlugin(context.Background(), PluginQuery{})
			return err
		}},
		{"empty capability ID", func() error {
			_, err := service.FindPluginsByCapability(context.Background(), CapabilityQuery{})
			return err
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, ErrInvalidQuery) {
				t.Fatalf("query error = %v, want ErrInvalidQuery", err)
			}
		})
	}
}

func TestDiscoveryPropagatesContextCancellation(t *testing.T) {
	service := newTestService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.ListPlugins(ctx, ListQuery{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ListPlugins() error = %v, want context.Canceled", err)
	}
}

func TestPageSizeIsBounded(t *testing.T) {
	size, err := normalizePageSize(1000)
	if err != nil {
		t.Fatalf("normalizePageSize() error = %v", err)
	}
	if size != int(maximumPageSize) {
		t.Fatalf("page size = %d, want %d", size, maximumPageSize)
	}
	if _, err := normalizePageSize(-1); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("negative page size error = %v", err)
	}
	if got := fmt.Sprint(defaultPageSize); got != "50" {
		t.Fatalf("default page size = %s", got)
	}
}
