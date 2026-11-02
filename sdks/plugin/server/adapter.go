package server

import (
	"context"
	"errors"
	"fmt"

	pluginv1 "pride-plugin-sdk/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Adapter struct {
	pluginv1.UnimplementedPluginServiceServer
	discovery DiscoveryPort
}

func New(discovery DiscoveryPort) (*Adapter, error) {
	if discovery == nil { return nil, fmt.Errorf("plugin sdk server: discovery port is nil") }
	return &Adapter{discovery: discovery}, nil
}

func (a *Adapter) ListPlugins(ctx context.Context, req *pluginv1.ListPluginsRequest) (*pluginv1.ListPluginsResponse, error) {
	if req == nil { return nil, invalid("", "request is required") }
	if err := validateRequestID(req.GetRequestId()); err != nil { return nil, err }
	if req.GetPageSize() < 0 { return nil, invalid(req.GetRequestId(), "page_size must not be negative") }
	meta, err := decodeStruct(req.GetMetadata())
	if err != nil { return nil, invalid(req.GetRequestId(), "metadata is invalid") }
	page, err := a.discovery.ListPlugins(ctx, ListQuery{PageSize:req.GetPageSize(), PageToken:req.GetPageToken(), Type:fromProtoType(req.GetTypeFilter()), State:fromProtoState(req.GetStateFilter()), Metadata:meta})
	if err != nil { return nil, toStatus(req.GetRequestId(), err) }
	plugins, err := mapPlugins(page.Plugins)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	responseMeta, err := encodeStruct(page.ResponseMetadata)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	return &pluginv1.ListPluginsResponse{RequestId:req.GetRequestId(), Plugins:plugins, NextPageToken:page.NextPageToken, Metadata:responseMeta}, nil
}

func (a *Adapter) GetPlugin(ctx context.Context, req *pluginv1.GetPluginRequest) (*pluginv1.GetPluginResponse, error) {
	if req == nil { return nil, invalid("", "request is required") }
	if err := validateRequestID(req.GetRequestId()); err != nil { return nil, err }
	if req.GetPluginId() == "" { return nil, invalid(req.GetRequestId(), "plugin_id is required") }
	plugin, err := a.discovery.GetPlugin(ctx, req.GetPluginId())
	if err != nil { return nil, toStatus(req.GetRequestId(), err) }
	mapped, err := mapPlugin(plugin)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	meta, err := responseMetadata(nil)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	return &pluginv1.GetPluginResponse{RequestId:req.GetRequestId(), Plugin:mapped, Metadata:meta}, nil
}

func (a *Adapter) FindPluginsByCapability(ctx context.Context, req *pluginv1.FindPluginsByCapabilityRequest) (*pluginv1.FindPluginsByCapabilityResponse, error) {
	if req == nil { return nil, invalid("", "request is required") }
	if err := validateRequestID(req.GetRequestId()); err != nil { return nil, err }
	if req.GetCapabilityId() == "" { return nil, invalid(req.GetRequestId(), "capability_id is required") }
	if req.GetPageSize() < 0 { return nil, invalid(req.GetRequestId(), "page_size must not be negative") }
	meta, err := decodeStruct(req.GetMetadata())
	if err != nil { return nil, invalid(req.GetRequestId(), "metadata is invalid") }
	page, err := a.discovery.FindPluginsByCapability(ctx, CapabilityQuery{CapabilityID:req.GetCapabilityId(), PageSize:req.GetPageSize(), PageToken:req.GetPageToken(), Metadata:meta})
	if err != nil { return nil, toStatus(req.GetRequestId(), err) }
	plugins, err := mapPlugins(page.Plugins)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	responseMeta, err := encodeStruct(page.ResponseMetadata)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	return &pluginv1.FindPluginsByCapabilityResponse{RequestId:req.GetRequestId(), Plugins:plugins, NextPageToken:page.NextPageToken, Metadata:responseMeta}, nil
}

func (a *Adapter) GetPluginMetadata(ctx context.Context, req *pluginv1.GetPluginMetadataRequest) (*pluginv1.GetPluginMetadataResponse, error) {
	if req == nil { return nil, invalid("", "request is required") }
	if err := validateRequestID(req.GetRequestId()); err != nil { return nil, err }
	if req.GetPluginId() == "" { return nil, invalid(req.GetRequestId(), "plugin_id is required") }
	value, err := a.discovery.GetPluginMetadata(ctx, req.GetPluginId())
	if err != nil { return nil, toStatus(req.GetRequestId(), err) }
	caps, err := mapCapabilities(value.Capabilities)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	attrs, err := encodeStruct(value.Attributes)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	meta, err := encodeStruct(value.ResponseMetadata)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	return &pluginv1.GetPluginMetadataResponse{RequestId:req.GetRequestId(), PluginMetadata:&pluginv1.PluginMetadata{PluginId:value.PluginID, Name:value.Name, Version:value.Version, Type:toProtoType(value.Type), Capabilities:caps, PermissionIds:append([]string(nil), value.PermissionIDs...), CompatibilityStatus:value.CompatibilityStatus, Attributes:attrs}, Metadata:meta}, nil
}

func (a *Adapter) GetPluginState(ctx context.Context, req *pluginv1.GetPluginStateRequest) (*pluginv1.GetPluginStateResponse, error) {
	if req == nil { return nil, invalid("", "request is required") }
	if err := validateRequestID(req.GetRequestId()); err != nil { return nil, err }
	if req.GetPluginId() == "" { return nil, invalid(req.GetRequestId(), "plugin_id is required") }
	value, err := a.discovery.GetPluginState(ctx, req.GetPluginId())
	if err != nil { return nil, toStatus(req.GetRequestId(), err) }
	meta, err := encodeStruct(value.ResponseMetadata)
	if err != nil { return nil, internal(req.GetRequestId(), err) }
	return &pluginv1.GetPluginStateResponse{RequestId:req.GetRequestId(), PluginId:value.PluginID, State:toProtoState(value.State), Available:value.Available, StatusMessage:value.StatusMessage, Metadata:meta}, nil
}

func validateRequestID(id string) error {
	if id == "" { return invalid(id, "request_id is required") }
	return nil
}
func invalid(id, message string) error { return buildStatus(codes.InvalidArgument, id, string(CodeInvalidRequest), message) }
func internal(id string, err error) error { return buildStatus(codes.Internal, id, string(CodeInternal), "Plugin discovery failed") }
func toStatus(id string, err error) error {
	var domain *DomainError
	if !errors.As(err, &domain) { return internal(id, err) }
	code := codes.Internal
	switch domain.Code {
	case CodeInvalidRequest, CodeInvalidInput: code = codes.InvalidArgument
	case CodePluginNotFound, CodeCapabilityNotFound: code = codes.NotFound
	case CodePluginUnavailable: code = codes.Unavailable
	case CodePermissionDenied: code = codes.PermissionDenied
	case CodeInternal: code = codes.Internal
	default: return internal(id, err)
	}
	message := domain.Message
	if message == "" { message = "Plugin discovery failed" }
	return buildStatus(code, id, string(domain.Code), message)
}
func buildStatus(code codes.Code, id, errorCode, message string) error {
	detail := &pluginv1.PluginError{RequestId:id, ErrorCode:errorCode, Message:message}
	st, err := status.New(code, message).WithDetails(detail)
	if err != nil { return status.Error(code, message) }
	return st.Err()
}
func decodeStruct(value *structpb.Struct) (map[string]any, error) {
	if value == nil { return nil, nil }
	return value.AsMap(), nil
}
func encodeStruct(value map[string]any) (*structpb.Struct, error) {
	if len(value) == 0 { return nil, nil }
	return structpb.NewStruct(value)
}
func responseMetadata(value map[string]any) (*structpb.Struct, error) { return encodeStruct(value) }

func mapPlugins(values []Plugin) ([]*pluginv1.PluginRecord, error) {
	out := make([]*pluginv1.PluginRecord, 0, len(values))
	for _, value := range values { mapped, err := mapPlugin(value); if err != nil { return nil, err }; out = append(out, mapped) }
	return out, nil
}
func mapPlugin(value Plugin) (*pluginv1.PluginRecord, error) {
	caps, err := mapCapabilities(value.Capabilities); if err != nil { return nil, err }
	return &pluginv1.PluginRecord{Manifest:mapManifest(value.Manifest), Type:toProtoType(value.Type), State:toProtoState(value.State), Capabilities:caps, Available:value.Available, CompatibilityStatus:value.CompatibilityStatus}, nil
}
func mapManifest(value Manifest) *pluginv1.PluginManifest {
	return &pluginv1.PluginManifest{PluginId:value.PluginID, Name:value.Name, Version:value.Version, Description:value.Description, Author:value.Author, License:value.License, EntryPoint:value.EntryPoint, MinimumPrideVersion:value.MinimumPRIDEVersion, Homepage:value.Homepage, DocumentationUrl:value.DocumentationURL, RepositoryUrl:value.RepositoryURL, IssueTrackerUrl:value.IssueTrackerURL, Keywords:append([]string(nil), value.Keywords...), Category:value.Category, Icon:value.Icon, ChangelogUrl:value.ChangelogURL}
}
func mapCapabilities(values []Capability) ([]*pluginv1.PluginCapability, error) {
	out := make([]*pluginv1.PluginCapability, 0, len(values))
	for _, value := range values { out = append(out, &pluginv1.PluginCapability{CapabilityId:value.ID, Name:value.Name, Description:value.Description}) }
	return out, nil
}
func fromProtoType(value pluginv1.PluginType) PluginType { return PluginType(value) }
func toProtoType(value PluginType) pluginv1.PluginType { if value < PluginTypeUnspecified || value > PluginTypePassive { return pluginv1.PluginType_PLUGIN_TYPE_UNSPECIFIED }; return pluginv1.PluginType(value) }
func fromProtoState(value pluginv1.PluginState) PluginState { return PluginState(value) }
func toProtoState(value PluginState) pluginv1.PluginState { if value < PluginStateUnspecified || value > PluginStateError { return pluginv1.PluginState_PLUGIN_STATE_UNSPECIFIED }; return pluginv1.PluginState(value) }

var _ pluginv1.PluginServiceServer = (*Adapter)(nil)
var _ = context.Canceled
