package server

import (
	"context"
	"testing"

	pluginv1 "pride-plugin-sdk/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestListPluginsRejectsUnknownEnumFiltersBeforeCallingPort(t *testing.T) {
	fake := &fakeDiscovery{}
	adapter, err := New(fake)
	if err != nil {
		t.Fatal(err)
	}

	_, err = adapter.ListPlugins(context.Background(), &pluginv1.ListPluginsRequest{
		RequestId: "req-invalid-type",
		PageSize:  12,
		TypeFilter: pluginv1.PluginType(99),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", status.Code(err))
	}
	if fake.listQuery.PageSize != 0 {
		t.Fatalf("discovery port was called for invalid enum: %+v", fake.listQuery)
	}

	_, err = adapter.ListPlugins(context.Background(), &pluginv1.ListPluginsRequest{
		RequestId:  "req-invalid-state",
		PageSize:   12,
		StateFilter: pluginv1.PluginState(-1),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %v, want InvalidArgument", status.Code(err))
	}
	if fake.listQuery.PageSize != 0 {
		t.Fatalf("discovery port was called for invalid state enum: %+v", fake.listQuery)
	}
}

type invalidMetadataDiscovery struct {
	*fakeDiscovery
}

func (d *invalidMetadataDiscovery) GetPluginMetadata(context.Context, PluginQuery) (PluginMetadata, error) {
	return PluginMetadata{PluginID: "example", Type: PluginType(99)}, nil
}

func TestAdapterRejectsInvalidModuleEnums(t *testing.T) {
	tests := []struct {
		name string
		call func(*Adapter) error
	}{
		{
			name: "plugin record type",
			call: func(adapter *Adapter) error {
				_, err := adapter.GetPlugin(context.Background(), &pluginv1.GetPluginRequest{
					RequestId: "req-invalid-module-type", PluginId: "example",
				})
				return err
			},
		},
		{
			name: "plugin record state",
			call: func(adapter *Adapter) error {
				_, err := adapter.GetPluginState(context.Background(), &pluginv1.GetPluginStateRequest{
					RequestId: "req-invalid-module-state", PluginId: "example",
				})
				return err
			},
		},
		{
			name: "metadata type",
			call: func(adapter *Adapter) error {
				_, err := adapter.GetPluginMetadata(context.Background(), &pluginv1.GetPluginMetadataRequest{
					RequestId: "req-invalid-metadata-type", PluginId: "example",
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeDiscovery{
				plugin: Plugin{
					Manifest: Manifest{PluginID: "example"},
					Type: PluginType(99),
					State: PluginState(99),
				},
			}
			var discovery DiscoveryPort = fake
			if tt.name == "metadata type" {
				discovery = &invalidMetadataDiscovery{fakeDiscovery: fake}
			}
			adapter, err := New(discovery)
			if err != nil {
				t.Fatal(err)
			}
			err = tt.call(adapter)
			if status.Code(err) != codes.Internal {
				t.Fatalf("status = %v, want Internal", status.Code(err))
			}
		})
	}
}

func TestToStatusMapsDiscoveryErrors(t *testing.T) {
	tests := []struct {
		name string
		err error
		want codes.Code
		wantCode ErrorCode
	}{
		{"invalid input", &DomainError{Code: CodeInvalidInput}, codes.InvalidArgument, CodeInvalidInput},
		{"plugin missing", &DomainError{Code: CodePluginNotFound}, codes.NotFound, CodePluginNotFound},
		{"capability missing", &DomainError{Code: CodeCapabilityNotFound}, codes.NotFound, CodeCapabilityNotFound},
		{"unavailable", &DomainError{Code: CodePluginUnavailable}, codes.Unavailable, CodePluginUnavailable},
		{"not ready", &DomainError{Code: CodePluginNotReady}, codes.Unavailable, CodePluginNotReady},
		{"disabled", &DomainError{Code: CodePluginDisabled}, codes.FailedPrecondition, CodePluginDisabled},
		{"incompatible", &DomainError{Code: CodeIncompatiblePlugin}, codes.FailedPrecondition, CodeIncompatiblePlugin},
		{"permission denied", &DomainError{Code: CodePermissionDenied}, codes.PermissionDenied, CodePermissionDenied},
		{"timeout category", &DomainError{Code: CodeTimeout}, codes.DeadlineExceeded, CodeTimeout},
		{"lifecycle error", &DomainError{Code: CodeLifecycleError}, codes.Internal, CodeLifecycleError},
		{"execution error", &DomainError{Code: CodeExecutionError}, codes.Internal, CodeExecutionError},
		{"unknown category", &DomainError{Code: CodeUnknown}, codes.Internal, CodeUnknown},
		{"context canceled", context.Canceled, codes.Canceled, CodeUnknown},
		{"context deadline", context.DeadlineExceeded, codes.DeadlineExceeded, CodeTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := toStatus("req-error", tt.err)
			if got := status.Code(err); got != tt.want {
				t.Fatalf("status = %v, want %v", got, tt.want)
			}
			details := status.Convert(err).Details()
			if len(details) != 1 {
				t.Fatalf("details count = %d, want 1", len(details))
			}
			detail, ok := details[0].(*pluginv1.PluginError)
			if !ok {
				t.Fatalf("detail type = %T, want *PluginError", details[0])
			}
			if detail.GetRequestId() != "req-error" {
				t.Fatalf("request_id = %q, want req-error", detail.GetRequestId())
			}
			if detail.GetErrorCode() != string(tt.wantCode) {
				t.Fatalf("error_code = %q, want %q", detail.GetErrorCode(), tt.wantCode)
			}
		})
	}
}
