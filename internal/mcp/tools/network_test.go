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
			name: "Successful Flow Observation",
			input: ObserveNetworkFlowsInput{
				Namespace: "default",
				Pod:       "proxy",
				Last:      5,
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
			handler := NewObserveNetworkFlowsHandler(func(ctx context.Context, namespace, pod, reserved string, last int) (string, error) {
				if namespace != tt.input.Namespace {
					t.Errorf("got namespace %q, want %q", namespace, tt.input.Namespace)
				}
				if pod != tt.input.Pod {
					t.Errorf("got pod %q, want %q", pod, tt.input.Pod)
				}
				if reserved != tt.input.Reserved {
					t.Errorf("got reserved %q, want %q", reserved, tt.input.Reserved)
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
