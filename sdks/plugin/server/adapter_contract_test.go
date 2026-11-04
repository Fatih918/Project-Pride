package server

import (
	"context"
	"net"
	"testing"

	pluginv1 "pride-plugin-sdk/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/structpb"
)

type contractDiscovery struct {
	pluginQuery     PluginQuery
	listQuery       ListQuery
	capabilityQuery CapabilityQuery
	calls           []string
}

func (d *contractDiscovery) ListPlugins(_ context.Context, query ListQuery) (PluginPage, error) {
	d.calls = append(d.calls, "list")
	d.listQuery = query
	return PluginPage{
		Plugins:          []Plugin{contractPlugin()},
		NextPageToken:    "list-next",
		ResponseMetadata: map[string]any{"source": "list-registry"},
	}, nil
}

func (d *contractDiscovery) GetPlugin(_ context.Context, query PluginQuery) (Plugin, error) {
	d.calls = append(d.calls, "get")
	d.pluginQuery = query
	plugin := contractPlugin()
	plugin.ResponseMetadata = map[string]any{"source": "plugin-registry"}
	return plugin, nil
}

func (d *contractDiscovery) FindPluginsByCapability(_ context.Context, query CapabilityQuery) (PluginPage, error) {
	d.calls = append(d.calls, "find-capability")
	d.capabilityQuery = query
	return PluginPage{
		Plugins:          []Plugin{contractPlugin()},
		NextPageToken:    "capability-next",
		ResponseMetadata: map[string]any{"source": "capability-registry"},
	}, nil
}

func (d *contractDiscovery) GetPluginMetadata(_ context.Context, query PluginQuery) (PluginMetadata, error) {
	d.calls = append(d.calls, "metadata")
	d.pluginQuery = query
	return PluginMetadata{
		PluginID:            "example.plugin",
		Name:                "Example Plugin",
		Version:             "1.2.3",
		Type:                PluginTypeActive,
		Capabilities:        []Capability{{ID: "text.summarize", Name: "Summarize", Description: "Summarizes text"}},
		PermissionIDs:       []string{"document.read"},
		CompatibilityStatus: "compatible",
		Attributes:          map[string]any{"catalog": "official"},
		ResponseMetadata:    map[string]any{"source": "metadata-registry"},
	}, nil
}

func (d *contractDiscovery) GetPluginState(_ context.Context, query PluginQuery) (PluginStateInfo, error) {
	d.calls = append(d.calls, "state")
	d.pluginQuery = query
	return PluginStateInfo{
		PluginID:         "example.plugin",
		State:            PluginStateEnabled,
		Available:        true,
		StatusMessage:    "ready",
		ResponseMetadata: map[string]any{"source": "state-registry"},
	}, nil
}

func contractPlugin() Plugin {
	return Plugin{
		Manifest: Manifest{
			PluginID:            "example.plugin",
			Name:                "Example Plugin",
			Version:             "1.2.3",
			Description:         "Contract test plugin",
			Author:              "PRIDE",
			License:              "MIT",
			EntryPoint:           "main.go",
			MinimumPRIDEVersion:  "1.0.0",
			Keywords:             []string{"example", "contract"},
		},
		Type:                PluginTypeActive,
		State:               PluginStateEnabled,
		Capabilities:        []Capability{{ID: "text.summarize", Name: "Summarize", Description: "Summarizes text"}},
		Available:           true,
		CompatibilityStatus: "compatible",
	}
}

func newContractGRPCClient(t *testing.T, discovery DiscoveryPort) (pluginv1.PluginServiceClient, func()) {
	t.Helper()

	adapter, err := New(discovery)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pluginv1.RegisterPluginServiceServer(grpcServer, adapter)
	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///plugin-contract",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		grpcServer.Stop()
		_ = listener.Close()
		t.Fatalf("grpc.NewClient() error = %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}
	return pluginv1.NewPluginServiceClient(conn), cleanup
}

func contractRequestMetadata(t *testing.T) *structpb.Struct {
	t.Helper()
	value, err := structpb.NewStruct(map[string]any{"trace": "trace-123"})
	if err != nil {
		t.Fatalf("NewStruct() error = %v", err)
	}
	return value
}

func assertResponseSource(t *testing.T, metadata *structpb.Struct, want string) {
	t.Helper()
	if metadata == nil || metadata.GetFields()["source"].GetStringValue() != want {
		t.Fatalf("response metadata = %v, want source %q", metadata, want)
	}
}

func TestDiscoveryRPCsPreserveContractAcrossGRPC(t *testing.T) {
	discovery := &contractDiscovery{}
	client, cleanup := newContractGRPCClient(t, discovery)
	defer cleanup()

	ctx := context.Background()
	requestMetadata := contractRequestMetadata(t)

	t.Run("ListPlugins", func(t *testing.T) {
		response, err := client.ListPlugins(ctx, &pluginv1.ListPluginsRequest{
			RequestId: "req-list",
			PageSize:  25,
			PageToken: "list-current",
			TypeFilter: pluginv1.PluginType_PLUGIN_TYPE_ACTIVE,
			StateFilter: pluginv1.PluginState_PLUGIN_STATE_ENABLED,
			Metadata: requestMetadata,
		})
		if err != nil {
			t.Fatalf("ListPlugins() error = %v", err)
		}
		if response.GetRequestId() != "req-list" || response.GetNextPageToken() != "list-next" {
			t.Fatalf("ListPlugins() response identity/pagination = %+v", response)
		}
		if len(response.GetPlugins()) != 1 || response.GetPlugins()[0].GetManifest().GetPluginId() != "example.plugin" {
			t.Fatalf("ListPlugins() plugin mapping = %+v", response.GetPlugins())
		}
		assertResponseSource(t, response.GetMetadata(), "list-registry")
		if discovery.listQuery.PageSize != 25 || discovery.listQuery.PageToken != "list-current" ||
			discovery.listQuery.Type != PluginTypeActive || discovery.listQuery.State != PluginStateEnabled ||
			discovery.listQuery.Metadata["trace"] != "trace-123" {
			t.Fatalf("ListPlugins() query forwarding = %+v", discovery.listQuery)
		}
	})

	t.Run("GetPlugin", func(t *testing.T) {
		response, err := client.GetPlugin(ctx, &pluginv1.GetPluginRequest{
			RequestId: "req-get", PluginId: "example.plugin", Metadata: requestMetadata,
		})
		if err != nil {
			t.Fatalf("GetPlugin() error = %v", err)
		}
		if response.GetRequestId() != "req-get" || response.GetPlugin().GetManifest().GetPluginId() != "example.plugin" {
			t.Fatalf("GetPlugin() response = %+v", response)
		}
		assertResponseSource(t, response.GetMetadata(), "plugin-registry")
		if discovery.pluginQuery.PluginID != "example.plugin" || discovery.pluginQuery.Metadata["trace"] != "trace-123" {
			t.Fatalf("GetPlugin() query forwarding = %+v", discovery.pluginQuery)
		}
	})

	t.Run("FindPluginsByCapability", func(t *testing.T) {
		response, err := client.FindPluginsByCapability(ctx, &pluginv1.FindPluginsByCapabilityRequest{
			RequestId: "req-capability",
			CapabilityId: "text.summarize",
			PageSize:  10,
			PageToken: "capability-current",
			Metadata: requestMetadata,
		})
		if err != nil {
			t.Fatalf("FindPluginsByCapability() error = %v", err)
		}
		if response.GetRequestId() != "req-capability" || response.GetNextPageToken() != "capability-next" ||
			len(response.GetPlugins()) != 1 {
			t.Fatalf("FindPluginsByCapability() response = %+v", response)
		}
		assertResponseSource(t, response.GetMetadata(), "capability-registry")
		if discovery.capabilityQuery.CapabilityID != "text.summarize" ||
			discovery.capabilityQuery.PageSize != 10 ||
			discovery.capabilityQuery.PageToken != "capability-current" ||
			discovery.capabilityQuery.Metadata["trace"] != "trace-123" {
			t.Fatalf("FindPluginsByCapability() query forwarding = %+v", discovery.capabilityQuery)
		}
	})

	t.Run("GetPluginMetadata", func(t *testing.T) {
		response, err := client.GetPluginMetadata(ctx, &pluginv1.GetPluginMetadataRequest{
			RequestId: "req-metadata", PluginId: "example.plugin", Metadata: requestMetadata,
		})
		if err != nil {
			t.Fatalf("GetPluginMetadata() error = %v", err)
		}
		value := response.GetPluginMetadata()
		if response.GetRequestId() != "req-metadata" || value.GetPluginId() != "example.plugin" ||
			value.GetType() != pluginv1.PluginType_PLUGIN_TYPE_ACTIVE ||
			len(value.GetCapabilities()) != 1 || value.GetPermissionIds()[0] != "document.read" {
			t.Fatalf("GetPluginMetadata() response = %+v", response)
		}
		if value.GetAttributes().GetFields()["catalog"].GetStringValue() != "official" {
			t.Fatalf("metadata attributes = %v", value.GetAttributes())
		}
		assertResponseSource(t, response.GetMetadata(), "metadata-registry")
		if discovery.pluginQuery.PluginID != "example.plugin" || discovery.pluginQuery.Metadata["trace"] != "trace-123" {
			t.Fatalf("GetPluginMetadata() query forwarding = %+v", discovery.pluginQuery)
		}
	})

	t.Run("GetPluginState", func(t *testing.T) {
		response, err := client.GetPluginState(ctx, &pluginv1.GetPluginStateRequest{
			RequestId: "req-state", PluginId: "example.plugin", Metadata: requestMetadata,
		})
		if err != nil {
			t.Fatalf("GetPluginState() error = %v", err)
		}
		if response.GetRequestId() != "req-state" || response.GetPluginId() != "example.plugin" ||
			response.GetState() != pluginv1.PluginState_PLUGIN_STATE_ENABLED ||
			!response.GetAvailable() || response.GetStatusMessage() != "ready" {
			t.Fatalf("GetPluginState() response = %+v", response)
		}
		assertResponseSource(t, response.GetMetadata(), "state-registry")
		if discovery.pluginQuery.PluginID != "example.plugin" || discovery.pluginQuery.Metadata["trace"] != "trace-123" {
			t.Fatalf("GetPluginState() query forwarding = %+v", discovery.pluginQuery)
		}
	})

	if len(discovery.calls) != 5 {
		t.Fatalf("discovery calls = %v, want all five RPCs", discovery.calls)
	}
}
