package analytics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
)

type MockRunner struct {
	RunFn func(ctx context.Context, name string, arg ...string) ([]byte, error)
}

func (m *MockRunner) Run(ctx context.Context, name string, arg ...string) ([]byte, error) {
	if m.RunFn != nil {
		return m.RunFn(ctx, name, arg...)
	}
	return nil, nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newInMemoryHTTPClient routes requests directly into a handler without opening a TCP listener.
// This is needed in sandboxed environments where `httptest.NewServer` is not permitted.
func newInMemoryHTTPClient(h http.Handler) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			done := make(chan *http.Response, 1)
			go func() {
				rr := httptest.NewRecorder()
				r2 := req.Clone(req.Context())
				r2.RequestURI = r2.URL.RequestURI()
				h.ServeHTTP(rr, r2)
				resp := rr.Result()
				resp.Request = req
				done <- resp
			}()

			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case resp := <-done:
				return resp, nil
			}
		}),
	}
}

func TestThanosClient_QueryRange(t *testing.T) {
	tests := []struct {
		name         string
		responseJSON string
		status       int
		wantErr      bool
		wantCount    int
	}{
		{
			name: "Success",
			responseJSON: `{
				"status": "success",
				"data": {
					"resultType": "matrix",
					"result": [
						{
							"metric": { "instance": "host1" },
							"values": [
								[ 1708531200, "0.5" ]
							]
						}
					]
				}
			}`,
			status:    http.StatusOK,
			wantErr:   false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.responseJSON)
			})

			client := NewThanosClient("http://thanos")
			client.HTTPClient = newInMemoryHTTPClient(h)
			samples, err := client.QueryRange(context.Background(), "test", time.Now(), time.Now(), "1m")
			if (err != nil) != tt.wantErr {
				t.Fatalf("QueryRange() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(samples) != tt.wantCount {
				t.Errorf("expected %d samples, got %d", tt.wantCount, len(samples))
			}
		})
	}
}

func TestGetFunnelStatus(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantActive bool
		wantTarget string
	}{
		{
			name: "Active Funnel",
			mockOutput: `
https://server.ts.net:8443 (Funnel on)
|-- / proxy http://127.0.0.1:8085
`,
			mockErr:    nil,
			wantActive: true,
			wantTarget: "https://server.ts.net:8443",
		},
		{
			name:       "Inactive Funnel",
			mockOutput: "Funnel is off",
			mockErr:    errors.New("exit status 1"),
			wantActive: false,
			wantTarget: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner = &MockRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					return []byte(tt.mockOutput), tt.mockErr
				},
			}
			status, err := GetFunnelStatus(context.Background())
			if err != nil {
				t.Errorf("GetFunnelStatus() error = %v", err)
			}
			if status.Active != tt.wantActive {
				t.Errorf("Active = %v, want %v", status.Active, tt.wantActive)
			}
			if status.Target != tt.wantTarget {
				t.Errorf("Target = %v, want %v", status.Target, tt.wantTarget)
			}
		})
	}
}

func TestGetTailscaleStatus(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
	}{
		{
			name:       "Success",
			mockOutput: `{"BackendState": "Running"}`,
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "Command Error",
			mockOutput: "",
			mockErr:    errors.New("failed"),
			wantErr:    true,
		},
		{
			name:       "Invalid JSON",
			mockOutput: "invalid",
			mockErr:    nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner = &MockRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					return []byte(tt.mockOutput), tt.mockErr
				},
			}
			_, err := GetTailscaleStatus(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTailscaleStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestThanosClient_QueryRange_Errors(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantErr        bool
	}{
		{
			name:           "HTTP Error",
			serverResponse: "Internal Server Error",
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "Malformed Response Status",
			serverResponse: `{"status": "error"}`,
			serverStatus:   http.StatusOK,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				fmt.Fprint(w, tt.serverResponse)
			})

			client := NewThanosClient("http://thanos")
			client.HTTPClient = newInMemoryHTTPClient(h)
			_, err := client.QueryRange(context.Background(), "test", time.Now(), time.Now(), "1m")
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestThanosResourceProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("GetEnergyJoules", func(t *testing.T) {
		tests := []struct {
			name     string
			response string
			want     float64
			wantErr  bool
		}{
			{
				name: "Success",
				response: `{
					"status": "success",
					"data": {
						"resultType": "matrix",
						"result": [{"metric": {}, "values": [[1708531200, "123.45"]]}]
					}
				}`,
				want: 123.45,
			},
			{
				name:     "No Samples",
				response: `{"status": "success", "data": {"resultType": "matrix", "result": []}}`,
				want:     0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprint(w, tt.response)
				})
				client := NewThanosClient("http://thanos")
				client.HTTPClient = newInMemoryHTTPClient(h)
				provider := NewThanosResourceProvider(client)

				val, err := provider.GetEnergyJoules(ctx, time.Now(), time.Now())
				if (err != nil) != tt.wantErr {
					t.Errorf("GetEnergyJoules() error = %v, wantErr %v", err, tt.wantErr)
				}
				if val != tt.want {
					t.Errorf("got %v, want %v", val, tt.want)
				}
			})
		}
	})

	t.Run("GetContainerEnergy", func(t *testing.T) {
		tests := []struct {
			name     string
			response string
			want     map[string]float64
			wantErr  bool
		}{
			{
				name: "Success with Filter",
				response: `{
					"status": "success",
					"data": {
						"resultType": "matrix",
						"result": [
							{"metric": {"container_name": "pod1"}, "values": [[1708531200, "10.5"]]},
							{"metric": {"container_name": "POD"}, "values": [[1708531200, "1.0"]]}
						]
					}
				}`,
				want: map[string]float64{"pod1": 10.5},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprint(w, tt.response)
				})
				client := NewThanosClient("http://thanos")
				client.HTTPClient = newInMemoryHTTPClient(h)
				provider := NewThanosResourceProvider(client)

				res, err := provider.GetContainerEnergy(ctx, time.Now(), time.Now())
				if (err != nil) != tt.wantErr {
					t.Errorf("GetContainerEnergy() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !reflect.DeepEqual(res, tt.want) {
					t.Errorf("got %v, want %v", res, tt.want)
				}
			})
		}
	})

	t.Run("GetHostServiceCPU", func(t *testing.T) {
		tests := []struct {
			name     string
			response string
			want     map[string]float64
			wantErr  bool
		}{
			{
				name: "Success",
				response: `{
					"status": "success",
					"data": {
						"resultType": "matrix",
						"result": [{"metric": {"name": "proxy.service"}, "values": [[1708531200, "0.25"]]}]
					}
				}`,
				want: map[string]float64{"proxy.service": 0.25},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprint(w, tt.response)
				})
				client := NewThanosClient("http://thanos")
				client.HTTPClient = newInMemoryHTTPClient(h)
				provider := NewThanosResourceProvider(client)

				res, err := provider.GetHostServiceCPU(ctx, time.Now(), time.Now())
				if (err != nil) != tt.wantErr {
					t.Errorf("GetHostServiceCPU() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !reflect.DeepEqual(res, tt.want) {
					t.Errorf("got %v, want %v", res, tt.want)
				}
			})
		}
	})

	t.Run("GetValueUnits", func(t *testing.T) {
		tests := []struct {
			name     string
			response string
			want     map[string]float64
			wantErr  bool
		}{
			{
				name: "Success",
				response: `{
					"status": "success",
					"data": {
						"resultType": "matrix",
						"result": [{"metric": {}, "values": [[1708531200, "5"]]}]
					}
				}`,
				want: map[string]float64{"ingestion": 5, "proxy": 5},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprint(w, tt.response)
				})
				client := NewThanosClient("http://thanos")
				client.HTTPClient = newInMemoryHTTPClient(h)
				provider := NewThanosResourceProvider(client)

				res, err := provider.GetValueUnits(ctx, time.Now(), time.Now())
				if (err != nil) != tt.wantErr {
					t.Errorf("GetValueUnits() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !reflect.DeepEqual(res, tt.want) {
					t.Errorf("got %v, want %v", res, tt.want)
				}
			})
		}
	})

	t.Run("Factors", func(t *testing.T) {
		tests := []struct {
			name       string
			envCarbon  string
			envCost    string
			wantCarbon float64
			wantCost   float64
		}{
			{
				name:       "Custom Environment",
				envCarbon:  "200.0",
				envCost:    "0.20",
				wantCarbon: 200.0,
				wantCost:   0.20 / 3600000.0,
			},
			{
				name:       "Defaults",
				envCarbon:  "",
				envCost:    "",
				wantCarbon: 150.0,
				wantCost:   0.15 / 3600000.0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.envCarbon != "" {
					t.Setenv("CARBON_INTENSITY_G_KWH", tt.envCarbon)
				} else {
					os.Unsetenv("CARBON_INTENSITY_G_KWH")
				}
				if tt.envCost != "" {
					t.Setenv("ENERGY_COST_CAD_KWH", tt.envCost)
				} else {
					os.Unsetenv("ENERGY_COST_CAD_KWH")
				}

				provider := NewThanosResourceProvider(nil)
				carbon, _ := provider.GetCarbonIntensity(ctx)
				cost, _ := provider.GetCostFactor(ctx)

				if carbon != tt.wantCarbon {
					t.Errorf("carbon got %v, want %v", carbon, tt.wantCarbon)
				}
				if diff := cost - tt.wantCost; diff < -1e-15 || diff > 1e-15 {
					t.Errorf("cost got %v, want %v", cost, tt.wantCost)
				}
			})
		}
	})
}

func TestRealCommandRunner_Run(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		want    string
		wantErr bool
	}{
		{
			name:    "Echo Success",
			command: "echo",
			args:    []string{"hello"},
			want:    "hello\n",
			wantErr: false,
		},
	}

	r := &RealCommandRunner{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := r.Run(context.Background(), tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RealCommandRunner.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if string(out) != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, string(out))
			}
		})
	}
}
