package sdkadapter_test

import (
	"context"
	"net"
	"testing"

	"pride-plugin-module/discovery"
	"pride-plugin-module/domain"
	"pride-plugin-module/registry"
	pluginv1 "pride-plugin-sdk/proto"
	pluginserver "pride-plugin-sdk/server"
	"pride-plugin-module-sdkadapter"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufferSize = 1024 * 1024

func TestDiscoveryRPCsThroughBufconn(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	listed, err := client.ListPlugins(ctx, &pluginv1.ListPluginsRequest{
		RequestId: "request-list",
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("ListPlugins() error = %v", err)
	}
	if listed.GetRequestId() != "request-list" || len(listed.GetPlugins()) != 1 {
		t.Fatalf("ListPlugins() response = %#v", listed)
	}
	record := listed.GetPlugins()[0]
	if record.GetManifest().GetPluginId() != "github-integration" {
		t.Fatalf("ListPlugins() plugin ID = %q", record.GetManifest().GetPluginId())
	}
	if record.GetType() != pluginv1.PluginType_PLUGIN_TYPE_ACTIVE ||
		record.GetState() != pluginv1.PluginState_PLUGIN_STATE_ENABLED {
		t.Fatalf("ListPlugins() type/state = %v/%v", record.GetType(), record.GetState())
	}
	if len(record.GetCapabilities()) != 1 || record.GetCapabilities()[0].GetCapabilityId() != "text.generate" {
		t.Fatalf("ListPlugins() capabilities = %#v", record.GetCapabilities())
	}

	got, err := client.GetPlugin(ctx, &pluginv1.GetPluginRequest{
		RequestId: "request-get",
		PluginId:  "github-integration",
	})
	if err != nil {
		t.Fatalf("GetPlugin() error = %v", err)
	}
	if got.GetRequestId() != "request-get" ||
		got.GetPlugin().GetManifest().GetName() != "GitHub Integration" {
		t.Fatalf("GetPlugin() response = %#v", got)
	}

	matches, err := client.FindPluginsByCapability(ctx, &pluginv1.FindPluginsByCapabilityRequest{
		RequestId:    "request-capability",
		CapabilityId: "text.generate",
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("FindPluginsByCapability() error = %v", err)
	}
	if matches.GetRequestId() != "request-capability" || len(matches.GetPlugins()) != 1 {
		t.Fatalf("FindPluginsByCapability() response = %#v", matches)
	}

	metadata, err := client.GetPluginMetadata(ctx, &pluginv1.GetPluginMetadataRequest{
		RequestId: "request-metadata",
		PluginId:  "github-integration",
	})
	if err != nil {
		t.Fatalf("GetPluginMetadata() error = %v", err)
	}
	if metadata.GetRequestId() != "request-metadata" ||
		metadata.GetPluginMetadata().GetPluginId() != "github-integration" {
		t.Fatalf("GetPluginMetadata() response = %#v", metadata)
	}
	if len(metadata.GetPluginMetadata().GetPermissionIds()) != 1 ||
		metadata.GetPluginMetadata().GetPermissionIds()[0] != "network.read" {
		t.Fatalf("GetPluginMetadata() permission IDs = %v", metadata.GetPluginMetadata().GetPermissionIds())
	}
	if metadata.GetPluginMetadata().GetAttributes().AsMap()["publisher"] != "PRIDE Test Fixture" {
		t.Fatalf("GetPluginMetadata() attributes = %#v", metadata.GetPluginMetadata().GetAttributes())
	}

	state, err := client.GetPluginState(ctx, &pluginv1.GetPluginStateRequest{
		RequestId: "request-state",
		PluginId:  "github-integration",
	})
	if err != nil {
		t.Fatalf("GetPluginState() error = %v", err)
	}
	if state.GetRequestId() != "request-state" ||
		state.GetPluginId() != "github-integration" ||
		state.GetState() != pluginv1.PluginState_PLUGIN_STATE_ENABLED ||
		!state.GetAvailable() {
		t.Fatalf("GetPluginState() response = %#v", state)
	}
}

func TestDiscoveryErrorMappingThroughBufconn(t *testing.T) {
	client := newTestClient(t)

	_, err := client.GetPlugin(context.Background(), &pluginv1.GetPluginRequest{
		RequestId: "request-missing",
		PluginId:  "does-not-exist",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("GetPlugin() status = %v, want %v (err: %v)", status.Code(err), codes.NotFound, err)
	}

	grpcStatus := status.Convert(err)
	details := grpcStatus.Details()
	if len(details) != 1 {
		t.Fatalf("error details = %#v, want one PluginError", details)
	}
	pluginError, ok := details[0].(*pluginv1.PluginError)
	if !ok {
		t.Fatalf("error detail type = %T, want *pluginv1.PluginError", details[0])
	}
	if pluginError.GetRequestId() != "request-missing" ||
		pluginError.GetErrorCode() != "PLUGIN_NOT_FOUND" {
		t.Fatalf("PluginError detail = %#v", pluginError)
	}
}

func newTestClient(t *testing.T) pluginv1.PluginServiceClient {
	t.Helper()

	repository := registry.NewMemoryRepository()
	plugin := domain.Plugin{
		ID: "github-integration",
		Manifest: domain.Manifest{
			PluginID:            "github-integration",
			Name:                "GitHub Integration",
			Version:             "1.2.0",
			Description:         "Integration fixture for Plugin SDK discovery.",
			Author:              "PRIDE Test Fixture",
			License:             "MIT",
			EntryPoint:          "plugin",
			MinimumPRIDEVersion: "0.1.0",
			Homepage:            "https://example.invalid",
			Keywords:            []string{"github", "integration"},
			Category:            "developer-tools",
		},
		Type:  domain.PluginTypeActive,
		State: domain.PluginStateEnabled,
		Protected: domain.ProtectedMetadata{
			Capabilities: []domain.Capability{{
				ID:          "text.generate",
				Name:        "Text Generation",
				Description: "Generates text.",
			}},
			PermissionIDs:       []string{"network.read"},
			CompatibilityStatus: "compatible",
			Attributes: map[string]any{
				"publisher": "PRIDE Test Fixture",
			},
		},
		Available:     true,
		StatusMessage: "Plugin is ready",
	}
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatalf("Create() fixture error = %v", err)
	}

	service, err := discovery.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	port, err := sdkadapter.New(service)
	if err != nil {
		t.Fatalf("sdkadapter.New() error = %v", err)
	}
	adapter, err := pluginserver.New(port)
	if err != nil {
		t.Fatalf("pluginserver.New() error = %v", err)
	}

	listener := bufconn.Listen(bufferSize)
	grpcServer := grpc.NewServer()
	pluginv1.RegisterPluginServiceServer(grpcServer, adapter)
	go func() {
		_ = grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	ctx := context.Background()
	connection, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = connection.Close()
	})

	_ = ctx
	return pluginv1.NewPluginServiceClient(connection)
}
