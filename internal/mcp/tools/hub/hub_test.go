package hub

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"observability-hub/internal/mcp/providers"
)

func TestInspectPlatformHandler_Execute(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(ctx context.Context) (map[string]interface{}, error)
		wantErr bool
	}{
		{
			name: "Success",
			mockFn: func(ctx context.Context) (map[string]interface{}, error) {
				return map[string]interface{}{"status": "healthy"}, nil
			},
			wantErr: false,
		},
		{
			name: "Failure",
			mockFn: func(ctx context.Context) (map[string]interface{}, error) {
				return nil, errors.New("platform error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewInspectPlatformHandler(tt.mockFn)
			_, err := h.Execute(context.Background(), HubInput{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInspectHostHandler_Execute(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(ctx context.Context) (*providers.HostResource, error)
		wantErr bool
	}{
		{
			name: "Success",
			mockFn: func(ctx context.Context) (*providers.HostResource, error) {
				return &providers.HostResource{CPUUsage: "10%"}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewInspectHostHandler(tt.mockFn)
			_, err := h.Execute(context.Background(), HubInput{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListHostServicesHandler_Execute(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(ctx context.Context) ([]providers.ServiceStatus, error)
		wantErr bool
	}{
		{
			name: "Success",
			mockFn: func(ctx context.Context) ([]providers.ServiceStatus, error) {
				return []providers.ServiceStatus{{Name: "test.service"}}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewListHostServicesHandler(tt.mockFn)
			got, err := h.Execute(context.Background(), HubInput{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				res := got.([]providers.ServiceStatus)
				if len(res) != 1 {
					t.Errorf("expected 1 service, got %d", len(res))
				}
			}
		})
	}
}

func TestQueryServiceLogsHandler_Execute(t *testing.T) {
	tests := []struct {
		name    string
		input   HubInput
		mockFn  func(ctx context.Context, service string, since string) (string, error)
		wantErr bool
	}{
		{
			name:  "Success",
			input: HubInput{Service: "test.service", Since: "5m"},
			mockFn: func(ctx context.Context, service, since string) (string, error) {
				return "logs", nil
			},
			wantErr: false,
		},
		{
			name:  "Service Name Check",
			input: HubInput{Service: "proxy.service"},
			mockFn: func(ctx context.Context, service, since string) (string, error) {
				if service != "proxy.service" {
					return "", errors.New("wrong service")
				}
				return "ok", nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewQueryServiceLogsHandler(tt.mockFn)
			got, err := h.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, "logs") && tt.name == "Success" {
				t.Errorf("got %v, want logs", got)
			}
		})
	}
}
