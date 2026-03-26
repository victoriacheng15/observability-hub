package tools

import (
	"context"
	"errors"
	"testing"
)

func TestObserveNetworkFlowsHandler_Execute(t *testing.T) {
	tests := []struct {
		name       string
		input      ObserveNetworkFlowsInput
		mockOutput string
		mockErr    error
		wantErr    bool
	}{
		{
			name: "Successful Flow Observation with All Filters",
			input: ObserveNetworkFlowsInput{
				Namespace:  "default",
				Pod:        "proxy",
				FromPod:    "frontend",
				ToPod:      "backend",
				Protocol:   "tcp",
				Port:       80,
				ToPort:     8080,
				Verdict:    "FORWARDED",
				HTTPStatus: "200",
				HTTPMethod: "GET",
				HTTPPath:   "/api/v1",
				Reserved:   "world",
				Last:       5,
			},
			mockOutput: `{"flow":{"verdict":"FORWARDED"}}`,
			wantErr:    false,
		},
		{
			name: "Provider Error",
			input: ObserveNetworkFlowsInput{
				Namespace: "default",
			},
			mockErr: errors.New("hubble unreachable"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewObserveNetworkFlowsHandler(func(ctx context.Context, namespace, pod, fromPod, toPod, protocol, verdict, httpStatus, httpMethod, httpPath, reserved string, port, toPort, last int) (string, error) {
				if namespace != tt.input.Namespace {
					t.Errorf("got namespace %q, want %q", namespace, tt.input.Namespace)
				}
				if pod != tt.input.Pod {
					t.Errorf("got pod %q, want %q", pod, tt.input.Pod)
				}
				if fromPod != tt.input.FromPod {
					t.Errorf("got fromPod %q, want %q", fromPod, tt.input.FromPod)
				}
				if toPod != tt.input.ToPod {
					t.Errorf("got toPod %q, want %q", toPod, tt.input.ToPod)
				}
				if protocol != tt.input.Protocol {
					t.Errorf("got protocol %q, want %q", protocol, tt.input.Protocol)
				}
				if verdict != tt.input.Verdict {
					t.Errorf("got verdict %q, want %q", verdict, tt.input.Verdict)
				}
				if httpStatus != tt.input.HTTPStatus {
					t.Errorf("got httpStatus %q, want %q", httpStatus, tt.input.HTTPStatus)
				}
				if httpMethod != tt.input.HTTPMethod {
					t.Errorf("got httpMethod %q, want %q", httpMethod, tt.input.HTTPMethod)
				}
				if httpPath != tt.input.HTTPPath {
					t.Errorf("got httpPath %q, want %q", httpPath, tt.input.HTTPPath)
				}
				if reserved != tt.input.Reserved {
					t.Errorf("got reserved %q, want %q", reserved, tt.input.Reserved)
				}
				if port != tt.input.Port {
					t.Errorf("got port %d, want %d", port, tt.input.Port)
				}
				if toPort != tt.input.ToPort {
					t.Errorf("got toPort %d, want %d", toPort, tt.input.ToPort)
				}
				if last != tt.input.Last {
					t.Errorf("got last %d, want %d", last, tt.input.Last)
				}
				return tt.mockOutput, tt.mockErr
			})

			got, err := handler.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.mockOutput {
				t.Errorf("got %v, want %v", got, tt.mockOutput)
			}
		})
	}
}
