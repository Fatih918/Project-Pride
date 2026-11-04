package client

import (
	"context"
	"errors"
	"testing"

	pluginv1 "pride-plugin-sdk/proto"

	"google.golang.org/grpc"
)

// contractRPC captures the public client's forwarding behavior for every
// discovery method without requiring a network connection.
type contractRPC struct {
	pluginv1.PluginServiceClient
	calls       []string
	contexts    []context.Context
	requests    []any
	optionCount []int
	err         error
	nilResponse bool
}

func (f *contractRPC) record(method string, ctx context.Context, req any, opts []grpc.CallOption) {
	f.calls = append(f.calls, method)
	f.contexts = append(f.contexts, ctx)
	f.requests = append(f.requests, req)
	f.optionCount = append(f.optionCount, len(opts))
}

func (f *contractRPC) ListPlugins(ctx context.Context, req *pluginv1.ListPluginsRequest, opts ...grpc.CallOption) (*pluginv1.ListPluginsResponse, error) {
	f.record("ListPlugins", ctx, req, opts)
	if f.err != nil || f.nilResponse { return nil, f.err }
	return &pluginv1.ListPluginsResponse{RequestId: req.GetRequestId()}, nil
}

func (f *contractRPC) GetPlugin(ctx context.Context, req *pluginv1.GetPluginRequest, opts ...grpc.CallOption) (*pluginv1.GetPluginResponse, error) {
	f.record("GetPlugin", ctx, req, opts)
	if f.err != nil || f.nilResponse { return nil, f.err }
	return &pluginv1.GetPluginResponse{RequestId: req.GetRequestId()}, nil
}

func (f *contractRPC) FindPluginsByCapability(ctx context.Context, req *pluginv1.FindPluginsByCapabilityRequest, opts ...grpc.CallOption) (*pluginv1.FindPluginsByCapabilityResponse, error) {
	f.record("FindPluginsByCapability", ctx, req, opts)
	if f.err != nil || f.nilResponse { return nil, f.err }
	return &pluginv1.FindPluginsByCapabilityResponse{RequestId: req.GetRequestId()}, nil
}

func (f *contractRPC) GetPluginMetadata(ctx context.Context, req *pluginv1.GetPluginMetadataRequest, opts ...grpc.CallOption) (*pluginv1.GetPluginMetadataResponse, error) {
	f.record("GetPluginMetadata", ctx, req, opts)
	if f.err != nil || f.nilResponse { return nil, f.err }
	return &pluginv1.GetPluginMetadataResponse{RequestId: req.GetRequestId()}, nil
}

func (f *contractRPC) GetPluginState(ctx context.Context, req *pluginv1.GetPluginStateRequest, opts ...grpc.CallOption) (*pluginv1.GetPluginStateResponse, error) {
	f.record("GetPluginState", ctx, req, opts)
	if f.err != nil || f.nilResponse { return nil, f.err }
	return &pluginv1.GetPluginStateResponse{RequestId: req.GetRequestId()}, nil
}

func TestDiscoveryMethodsForwardRequestsContextsAndCallOptions(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "trace")
	option := grpc.WaitForReady(true)
	rpc := &contractRPC{}
	c := newWithRPC(rpc)

	requests := []any{
		&pluginv1.ListPluginsRequest{RequestId: "req-list"},
		&pluginv1.GetPluginRequest{RequestId: "req-get", PluginId: "plugin.test"},
		&pluginv1.FindPluginsByCapabilityRequest{RequestId: "req-cap", CapabilityId: "files.read"},
		&pluginv1.GetPluginMetadataRequest{RequestId: "req-meta", PluginId: "plugin.test"},
		&pluginv1.GetPluginStateRequest{RequestId: "req-state", PluginId: "plugin.test"},
	}
	invocations := []struct {
		name string
		call func(any) (string, error)
	}{
		{"ListPlugins", func(req any) (string, error) {
			res, err := c.ListPlugins(ctx, req.(*pluginv1.ListPluginsRequest), option)
			return res.GetRequestId(), err
		}},
		{"GetPlugin", func(req any) (string, error) {
			res, err := c.GetPlugin(ctx, req.(*pluginv1.GetPluginRequest), option)
			return res.GetRequestId(), err
		}},
		{"FindPluginsByCapability", func(req any) (string, error) {
			res, err := c.FindPluginsByCapability(ctx, req.(*pluginv1.FindPluginsByCapabilityRequest), option)
			return res.GetRequestId(), err
		}},
		{"GetPluginMetadata", func(req any) (string, error) {
			res, err := c.GetPluginMetadata(ctx, req.(*pluginv1.GetPluginMetadataRequest), option)
			return res.GetRequestId(), err
		}},
		{"GetPluginState", func(req any) (string, error) {
			res, err := c.GetPluginState(ctx, req.(*pluginv1.GetPluginStateRequest), option)
			return res.GetRequestId(), err
		}},
	}

	for i, invocation := range invocations {
		t.Run(invocation.name, func(t *testing.T) {
			requestID, err := invocation.call(requests[i])
			if err != nil {
				t.Fatalf("call error = %v", err)
			}
			if requestID == "" {
				t.Fatal("response request_id is empty")
			}
			if got := rpc.calls[i]; got != invocation.name {
				t.Fatalf("RPC method = %q, want %q", got, invocation.name)
			}
			if rpc.contexts[i] != ctx {
				t.Fatal("context was not forwarded unchanged")
			}
			if rpc.requests[i] != requests[i] {
				t.Fatal("request was not forwarded unchanged")
			}
			if rpc.optionCount[i] != 1 {
				t.Fatalf("call options forwarded = %d, want 1", rpc.optionCount[i])
			}
		})
	}
	if len(rpc.calls) != len(invocations) {
		t.Fatalf("RPC calls = %d, want %d", len(rpc.calls), len(invocations))
	}
}

func TestDiscoveryMethodsReturnTransportErrorsUnchanged(t *testing.T) {
	want := errors.New("transport failure")
	rpc := &contractRPC{err: want}
	c := newWithRPC(rpc)
	ctx := context.Background()

	calls := []struct {
		name string
		call func() error
	}{
		{"ListPlugins", func() error { _, err := c.ListPlugins(ctx, &pluginv1.ListPluginsRequest{RequestId: "r"}); return err }},
		{"GetPlugin", func() error { _, err := c.GetPlugin(ctx, &pluginv1.GetPluginRequest{RequestId: "r", PluginId: "p"}); return err }},
		{"FindPluginsByCapability", func() error { _, err := c.FindPluginsByCapability(ctx, &pluginv1.FindPluginsByCapabilityRequest{RequestId: "r", CapabilityId: "c"}); return err }},
		{"GetPluginMetadata", func() error { _, err := c.GetPluginMetadata(ctx, &pluginv1.GetPluginMetadataRequest{RequestId: "r", PluginId: "p"}); return err }},
		{"GetPluginState", func() error { _, err := c.GetPluginState(ctx, &pluginv1.GetPluginStateRequest{RequestId: "r", PluginId: "p"}); return err }},
	}

	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, want) {
				t.Fatalf("error = %v, want original transport error", err)
			}
		})
	}
}

func TestDiscoveryMethodsRejectNilResponses(t *testing.T) {
	rpc := &contractRPC{nilResponse: true}
	c := newWithRPC(rpc)
	ctx := context.Background()

	calls := []struct {
		name string
		call func() error
	}{
		{"ListPlugins", func() error { _, err := c.ListPlugins(ctx, &pluginv1.ListPluginsRequest{RequestId: "r"}); return err }},
		{"GetPlugin", func() error { _, err := c.GetPlugin(ctx, &pluginv1.GetPluginRequest{RequestId: "r", PluginId: "p"}); return err }},
		{"FindPluginsByCapability", func() error { _, err := c.FindPluginsByCapability(ctx, &pluginv1.FindPluginsByCapabilityRequest{RequestId: "r", CapabilityId: "c"}); return err }},
		{"GetPluginMetadata", func() error { _, err := c.GetPluginMetadata(ctx, &pluginv1.GetPluginMetadataRequest{RequestId: "r", PluginId: "p"}); return err }},
		{"GetPluginState", func() error { _, err := c.GetPluginState(ctx, &pluginv1.GetPluginStateRequest{RequestId: "r", PluginId: "p"}); return err }},
	}

	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, ErrEmptyResponse) {
				t.Fatalf("error = %v, want ErrEmptyResponse", err)
			}
		})
	}
}
