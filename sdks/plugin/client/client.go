// Package client provides the public Go client for the PRIDE Plugin SDK.
package client

import (
	"context"
	"errors"
	"fmt"

	pluginv1 "pride-plugin-sdk/proto"

	"google.golang.org/grpc"
)

var (
	// ErrInvalidArgument indicates that a required SDK request value is missing.
	ErrInvalidArgument = errors.New("plugin sdk: invalid argument")
	// ErrEmptyResponse indicates that the transport returned no response and no error.
	ErrEmptyResponse = errors.New("plugin sdk: empty response")
)

// Client is a thin, typed wrapper around the generated PluginService client.
// It does not select Plugins, evaluate permissions, or implement Plugin behavior.
type Client struct {
	rpc pluginv1.PluginServiceClient
}

// New creates a Plugin SDK client using an existing gRPC connection.
// The caller owns the connection and is responsible for closing it.
func New(conn grpc.ClientConnInterface) (*Client, error) {
	if conn == nil {
		return nil, fmt.Errorf("%w: gRPC connection is nil", ErrInvalidArgument)
	}
	return newWithRPC(pluginv1.NewPluginServiceClient(conn)), nil
}

func newWithRPC(rpc pluginv1.PluginServiceClient) *Client {
	return &Client{rpc: rpc}
}

func (c *Client) validate(ctx context.Context, requestID string) error {
	if c == nil || c.rpc == nil {
		return fmt.Errorf("%w: client is nil or uninitialized", ErrInvalidArgument)
	}
	if ctx == nil {
		return fmt.Errorf("%w: context is nil", ErrInvalidArgument)
	}
	if requestID == "" {
		return fmt.Errorf("%w: request_id is required", ErrInvalidArgument)
	}
	return nil
}

// ListPlugins returns a page of installed Plugin records.
func (c *Client) ListPlugins(ctx context.Context, req *pluginv1.ListPluginsRequest, opts ...grpc.CallOption) (*pluginv1.ListPluginsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidArgument)
	}
	if err := c.validate(ctx, req.GetRequestId()); err != nil {
		return nil, err
	}
	res, err := c.rpc.ListPlugins(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	return res, nil
}

// GetPlugin returns a Plugin record by its stable Plugin ID.
func (c *Client) GetPlugin(ctx context.Context, req *pluginv1.GetPluginRequest, opts ...grpc.CallOption) (*pluginv1.GetPluginResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidArgument)
	}
	if err := c.validate(ctx, req.GetRequestId()); err != nil {
		return nil, err
	}
	if req.GetPluginId() == "" {
		return nil, fmt.Errorf("%w: plugin_id is required", ErrInvalidArgument)
	}
	res, err := c.rpc.GetPlugin(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	return res, nil
}

// FindPluginsByCapability returns Plugins declaring the requested capability.
func (c *Client) FindPluginsByCapability(ctx context.Context, req *pluginv1.FindPluginsByCapabilityRequest, opts ...grpc.CallOption) (*pluginv1.FindPluginsByCapabilityResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidArgument)
	}
	if err := c.validate(ctx, req.GetRequestId()); err != nil {
		return nil, err
	}
	if req.GetCapabilityId() == "" {
		return nil, fmt.Errorf("%w: capability_id is required", ErrInvalidArgument)
	}
	res, err := c.rpc.FindPluginsByCapability(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	return res, nil
}

// GetPluginMetadata returns public metadata for a Plugin.
func (c *Client) GetPluginMetadata(ctx context.Context, req *pluginv1.GetPluginMetadataRequest, opts ...grpc.CallOption) (*pluginv1.GetPluginMetadataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidArgument)
	}
	if err := c.validate(ctx, req.GetRequestId()); err != nil {
		return nil, err
	}
	if req.GetPluginId() == "" {
		return nil, fmt.Errorf("%w: plugin_id is required", ErrInvalidArgument)
	}
	res, err := c.rpc.GetPluginMetadata(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	return res, nil
}

// GetPluginState returns the current state reported for a Plugin.
func (c *Client) GetPluginState(ctx context.Context, req *pluginv1.GetPluginStateRequest, opts ...grpc.CallOption) (*pluginv1.GetPluginStateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidArgument)
	}
	if err := c.validate(ctx, req.GetRequestId()); err != nil {
		return nil, err
	}
	if req.GetPluginId() == "" {
		return nil, fmt.Errorf("%w: plugin_id is required", ErrInvalidArgument)
	}
	res, err := c.rpc.GetPluginState(ctx, req, opts...)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	return res, nil
}
