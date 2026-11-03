package server

import (
	"context"
	"net"
	"testing"

	pluginv1 "pride-plugin-sdk/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fakeDiscovery struct {
	plugin    Plugin
	listQuery ListQuery
	ctx       context.Context
	err       error
}

func (f *fakeDiscovery) ListPlugins(ctx context.Context, q ListQuery) (PluginPage, error) {
	f.ctx = ctx
	f.listQuery = q
	if f.err != nil {
		return PluginPage{}, f.err
	}
	return PluginPage{Plugins: []Plugin{f.plugin}, NextPageToken: "next"}, nil
}
func (f *fakeDiscovery) GetPlugin(context.Context, PluginQuery) (Plugin, error) {
	return f.plugin, f.err
}
func (f *fakeDiscovery) FindPluginsByCapability(context.Context, CapabilityQuery) (PluginPage, error) {
	return PluginPage{Plugins: []Plugin{f.plugin}}, f.err
}
func (f *fakeDiscovery) GetPluginMetadata(context.Context, PluginQuery) (PluginMetadata, error) {
	return PluginMetadata{PluginID: f.plugin.Manifest.PluginID, Name: f.plugin.Manifest.Name}, f.err
}
func (f *fakeDiscovery) GetPluginState(context.Context, PluginQuery) (PluginStateInfo, error) {
	return PluginStateInfo{PluginID: f.plugin.Manifest.PluginID, State: f.plugin.State, Available: f.plugin.Available}, f.err
}

func TestAdapterListPluginsOverGRPC(t *testing.T) {
	fake := &fakeDiscovery{plugin: Plugin{Manifest: Manifest{PluginID: "example", Name: "Example"}, Type: PluginTypeActive, State: PluginStateEnabled, Available: true}}
	adapter, err := New(fake)
	if err != nil {
		t.Fatal(err)
	}
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pluginv1.RegisterPluginServiceServer(grpcServer, adapter)
	go func() { _ = grpcServer.Serve(listener) }()
	defer grpcServer.Stop()
	ctx := context.Background()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := pluginv1.NewPluginServiceClient(conn)
	response, err := client.ListPlugins(ctx, &pluginv1.ListPluginsRequest{RequestId: "req-123", PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetRequestId() != "req-123" || len(response.GetPlugins()) != 1 || response.GetNextPageToken() != "next" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if fake.listQuery.PageSize != 20 {
		t.Fatalf("page size = %d, want 20", fake.listQuery.PageSize)
	}
}

func TestAdapterRejectsInvalidRequestBeforePort(t *testing.T) {
	fake := &fakeDiscovery{}
	adapter, _ := New(fake)
	_, err := adapter.GetPlugin(context.Background(), &pluginv1.GetPluginRequest{RequestId: "req-1"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", status.Code(err))
	}
}

func TestAdapterMapsDomainErrorWithRequestID(t *testing.T) {
	fake := &fakeDiscovery{err: &DomainError{Code: CodePluginNotFound, Message: "plugin not found"}}
	adapter, _ := New(fake)
	_, err := adapter.GetPlugin(context.Background(), &pluginv1.GetPluginRequest{RequestId: "req-404", PluginId: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status = %v, want NotFound", status.Code(err))
	}
	details := status.Convert(err).Details()
	if len(details) != 1 {
		t.Fatalf("details count = %d, want 1", len(details))
	}
	detail, ok := details[0].(*pluginv1.PluginError)
	if !ok || detail.GetRequestId() != "req-404" || detail.GetErrorCode() != string(CodePluginNotFound) {
		t.Fatalf("unexpected error detail: %#v", details[0])
	}
}
