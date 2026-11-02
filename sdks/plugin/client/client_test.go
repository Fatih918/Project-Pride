package client

import (
	"context"
	"errors"
	"testing"

	pluginv1 "pride-plugin-sdk/proto"

	"google.golang.org/grpc"
)

type fakePluginService struct {
	pluginv1.PluginServiceClient
	listResponse *pluginv1.ListPluginsResponse
	listErr error
	listCalls int
	lastContext context.Context
}

func (f *fakePluginService) ListPlugins(ctx context.Context, req *pluginv1.ListPluginsRequest, _ ...grpc.CallOption) (*pluginv1.ListPluginsResponse, error) {
	f.listCalls++
	f.lastContext = ctx
	return f.listResponse, f.listErr
}

func TestListPluginsPreservesContextAndRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "trace")
	rpc := &fakePluginService{listResponse: &pluginv1.ListPluginsResponse{RequestId: "req-1"}}
	c := newWithRPC(rpc)

	got, err := c.ListPlugins(ctx, &pluginv1.ListPluginsRequest{RequestId: "req-1"})
	if err != nil {
		t.Fatalf("ListPlugins() error = %v", err)
	}
	if got.GetRequestId() != "req-1" {
		t.Fatalf("response request_id = %q, want req-1", got.GetRequestId())
	}
	if rpc.lastContext != ctx {
		t.Fatal("context was not forwarded unchanged")
	}
	if rpc.listCalls != 1 {
		t.Fatalf("RPC calls = %d, want 1", rpc.listCalls)
	}
}

func TestListPluginsRejectsMissingRequestID(t *testing.T) {
	rpc := &fakePluginService{}
	c := newWithRPC(rpc)

	_, err := c.ListPlugins(context.Background(), &pluginv1.ListPluginsRequest{})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
	if rpc.listCalls != 0 {
		t.Fatalf("RPC calls = %d, want 0", rpc.listCalls)
	}
}

func TestGetPluginRejectsMissingPluginID(t *testing.T) {
	c := newWithRPC(&fakePluginService{})

	_, err := c.GetPlugin(context.Background(), &pluginv1.GetPluginRequest{RequestId: "req-1"})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestListPluginsReturnsTransportErrorUnchanged(t *testing.T) {
	want := errors.New("transport failure")
	c := newWithRPC(&fakePluginService{listErr: want})

	_, err := c.ListPlugins(context.Background(), &pluginv1.ListPluginsRequest{RequestId: "req-1"})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want wrapped/original transport error", err)
	}
}

func TestListPluginsRejectsNilResponse(t *testing.T) {
	c := newWithRPC(&fakePluginService{})

	_, err := c.ListPlugins(context.Background(), &pluginv1.ListPluginsRequest{RequestId: "req-1"})
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatalf("error = %v, want ErrEmptyResponse", err)
	}
}
