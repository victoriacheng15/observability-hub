package analytics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newInMemoryHTTPClient(h http.Handler) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			resp := rr.Result()
			resp.Request = req
			return resp, nil
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
			name:         "Success",
			responseJSON: `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"instance":"h1"},"values":[[1708531200,"0.5"]]}]}}`,
			status:       http.StatusOK,
			wantErr:      false,
			wantCount:    1,
		},
		{
			name:         "API Error",
			responseJSON: `{"status":"error","error":"bad query"}`,
			status:       http.StatusOK,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				t.Errorf("got %d samples, want %d", len(samples), tt.wantCount)
			}
		})
	}
}

func TestThanosResourceProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("GetEnergyJoules", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1708531200,"100.5"]]}]}}`)
		})
		client := NewThanosClient("http://thanos")
		client.HTTPClient = newInMemoryHTTPClient(h)
		provider := NewThanosResourceProvider(client)

		val, err := provider.GetEnergyJoules(ctx, time.Now(), time.Now())
		if err != nil || val != 100.5 {
			t.Errorf("GetEnergyJoules() got %v, %v; want 100.5, nil", val, err)
		}
	})

	t.Run("GetContainerEnergy", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"container_name":"c1"},"values":[[1708531200,"10"]]}]}}`)
		})
		client := NewThanosClient("http://thanos")
		client.HTTPClient = newInMemoryHTTPClient(h)
		provider := NewThanosResourceProvider(client)

		res, _ := provider.GetContainerEnergy(ctx, time.Now(), time.Now())
		want := map[string]float64{"c1": 10}
		if !reflect.DeepEqual(res, want) {
			t.Errorf("got %v, want %v", res, want)
		}
	})

	t.Run("Factors", func(t *testing.T) {
		t.Setenv("CARBON_INTENSITY_G_KWH", "200.0")
		t.Setenv("ENERGY_COST_CAD_KWH", "0.20")
		provider := NewThanosResourceProvider(nil)
		carbon, _ := provider.GetCarbonIntensity(ctx)
		cost, _ := provider.GetCostFactor(ctx)

		if carbon != 200.0 {
			t.Errorf("carbon got %v, want 200", carbon)
		}
		wantCost := 0.20 / 3600000.0
		if diff := cost - wantCost; diff < -1e-15 || diff > 1e-15 {
			t.Errorf("cost got %v, want %v", cost, wantCost)
		}
	})
}
