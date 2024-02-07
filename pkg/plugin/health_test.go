package plugin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type mockHealthCheckClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (m *mockHealthCheckClient) Do(req *http.Request) (*http.Response, error) {
	return m.do(req)
}

type mockVectorService struct{}

func (m *mockVectorService) Search(ctx context.Context, collection string, query string, topK uint64, filter map[string]interface{}) ([]store.SearchResult, error) {
	return []store.SearchResult{{Payload: map[string]any{"a": "b"}, Score: 1.0}}, nil
}

func (m *mockVectorService) Health(ctx context.Context) error {
	return nil
}

func (m *mockVectorService) Cancel() {}

// TestCheckHealth tests CheckHealth calls, using backend.CheckHealthRequest and backend.CheckHealthResponse.
func TestCheckHealth(t *testing.T) {

	// Set up and run test cases
	for _, tc := range []struct {
		name     string
		settings backend.AppInstanceSettings
		hcClient healthCheckClient
		vService vector.Service

		expDetails healthCheckDetails
	}{
		{
			name: "everything disabled",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{},
			},
			expDetails: healthCheckDetails{
				Provider: providerHealthDetails{
					Error:  "No models are working",
					Models: map[string]modelHealth{},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "openai enabled",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{providerKey: "abcd1234"},
			},
			hcClient: &mockHealthCheckClient{
				do: func(req *http.Request) (*http.Response, error) {
					body, _ := io.ReadAll(req.Body)
					if strings.Contains(string(body), "gpt-4") {
						body := io.NopCloser(strings.NewReader(`{"error": "model does not exist"}`))
						return &http.Response{StatusCode: http.StatusNotFound, Body: body}, nil
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
				},
			},
			expDetails: healthCheckDetails{
				Provider: providerHealthDetails{
					Configured: true,
					OK:         true,
					Models: map[string]modelHealth{
						"gpt-3.5-turbo": {OK: true, Error: ""},
						"gpt-4":         {OK: false, Error: `unexpected status code: 404: {"error": "model does not exist"}`},
					},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "vector enabled, no openai",
			settings: backend.AppInstanceSettings{
				JSONData: json.RawMessage(`{
					"vector": {
						"enabled": true,
						"embed": {
							"type": "openai",
							"openai": {
								"url": "http://localhost:3000"
							}
						},
						"store": {
							"type": "qdrant",
							"qdrant": {
								"address": "localhost:6334"
							}
						}
					}
				}`),
				DecryptedSecureJSONData: map[string]string{},
			},
			vService: &mockVectorService{},
			expDetails: healthCheckDetails{
				Provider: providerHealthDetails{
					Error:  "No models are working",
					Models: map[string]modelHealth{},
				},
				Vector: vectorHealthDetails{
					Enabled: true,
					OK:      true,
				},
				Version: "unknown",
			},
		},
		{
			name: "vector enabled with openai",
			settings: backend.AppInstanceSettings{
				JSONData: json.RawMessage(`{
					"vector": {
						"enabled": true,
						"embed": {
							"type": "openai"
						},
						"store": {
							"type": "qdrant",
							"qdrant": {
								"address": "localhost:6334"
							}
						}
					}
				}`),
				DecryptedSecureJSONData: map[string]string{providerKey: "abcd1234"},
			},
			vService: &mockVectorService{},
			hcClient: &mockHealthCheckClient{
				do: func(req *http.Request) (*http.Response, error) {
					body, _ := io.ReadAll(req.Body)
					if strings.Contains(string(body), "gpt-4") {
						body := io.NopCloser(strings.NewReader(`{"error": "model does not exist"}`))
						return &http.Response{StatusCode: http.StatusNotFound, Body: body}, nil
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
				},
			},
			expDetails: healthCheckDetails{
				Provider: providerHealthDetails{
					Configured: true,
					OK:         true,
					Error:      "",
					Models: map[string]modelHealth{
						"gpt-3.5-turbo": {OK: true, Error: ""},
						"gpt-4":         {OK: false, Error: `unexpected status code: 404: {"error": "model does not exist"}`},
					},
				},
				Vector: vectorHealthDetails{
					Enabled: true,
					OK:      true,
				},
				Version: "unknown",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			// Initialize app
			inst, err := NewApp(ctx, tc.settings)
			if err != nil {
				t.Fatalf("new app: %s", err)
			}
			if inst == nil {
				t.Fatal("inst must not be nil")
			}
			app, ok := inst.(*App)
			if !ok {
				t.Fatal("inst must be of type *App")
			}
			app.healthCheckClient = tc.hcClient
			app.vectorService = tc.vService
			// Request by calling CheckHealth.
			resp, err := app.CheckHealth(ctx, &backend.CheckHealthRequest{
				PluginContext: backend.PluginContext{
					AppInstanceSettings: &tc.settings,
				},
			})
			if err != nil {
				t.Fatalf("CheckHealth error: %s", err)
			}
			if resp == nil {
				t.Fatal("no response received from CheckHealth")
			}
			var details healthCheckDetails
			if err = json.Unmarshal(resp.JSONDetails, &details); err != nil {
				t.Errorf("non-JSON response details (%s): %s", resp.JSONDetails, err)
			}
			if details.Provider.OK != tc.expDetails.Provider.OK ||
				details.Provider.Configured != tc.expDetails.Provider.Configured ||
				details.Provider.Error != tc.expDetails.Provider.Error {
				t.Errorf("OpenAI details should be %+v, got %+v", tc.expDetails.Provider, details.Provider)
			}
			for k, v := range tc.expDetails.Provider.Models {
				if details.Provider.Models[k] != v {
					t.Errorf("OpenAI model %s should be %+v, got %+v", k, v, details.Provider.Models[k])
				}
			}
			if details.Vector != tc.expDetails.Vector {
				t.Errorf("vector details should be %v, got %v", tc.expDetails.Vector, details.Vector)
			}
		})
	}
}
