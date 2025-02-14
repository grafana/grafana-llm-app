package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type mockVectorService struct{}

func (m *mockVectorService) Search(ctx context.Context, collection string, query string, topK uint64, filter map[string]interface{}) ([]store.SearchResult, error) {
	return []store.SearchResult{{Payload: map[string]any{"a": "b"}, Score: 1.0}}, nil
}

func (m *mockVectorService) Health(ctx context.Context) error {
	return nil
}

func (m *mockVectorService) Cancel() {}

type mockOpenAIHealthResponse struct {
	code int
	body string
}

func newMockOpenAIHealthServer(responses []mockOpenAIHealthResponse) *httptest.Server {
	i := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if i >= len(responses) {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(responses[i].code)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[i].body))
		i += 1
	}))
}

// TestCheckHealth tests CheckHealth calls, using backend.CheckHealthRequest and backend.CheckHealthResponse.
func TestCheckHealth(t *testing.T) {

	// Set up and run test cases
	for _, tc := range []struct {
		name     string
		settings backend.AppInstanceSettings
		vService vector.Service

		responses []mockOpenAIHealthResponse

		expDetails healthCheckDetails
	}{
		{
			name: "everything disabled",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{},
				JSONData: json.RawMessage(`{
					"openai": {
						"provider": "openai",
						"url": "%s"
					}
				}`),
			},
			expDetails: healthCheckDetails{
				OpenAI: openAIHealthDetails{
					Error:     "No functioning models are available",
					Models:    map[Model]openAIModelHealth{},
					Assistant: openAIModelHealth{OK: false, Error: "Assistant not available"},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "explicitly disabled",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{},
				JSONData: json.RawMessage(`{
					"disabled": true,
					"openai": {
						"url": "%s"
					}
				}`),
			},
			expDetails: healthCheckDetails{
				OpenAI: openAIHealthDetails{
					Configured: true,
					Error:      "LLM functionality is disabled",
					Models:     map[Model]openAIModelHealth{},
					Assistant:  openAIModelHealth{OK: false, Error: ""},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "openai enabled, has assistant support",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
				JSONData: json.RawMessage(`{
					"openai": {
						"provider": "openai",
						"url": "%s"
					}
				}`),
			},
			responses: []mockOpenAIHealthResponse{
				{code: http.StatusOK, body: "{}"},
				{code: http.StatusNotFound, body: `{"error": {"message": "model does not exist"}}`},
				{code: http.StatusOK, body: `{}`},
			},
			expDetails: healthCheckDetails{
				OpenAI: openAIHealthDetails{
					Configured: true,
					OK:         true,
					Models: map[Model]openAIModelHealth{
						ModelBase:  {OK: true, Error: ""},
						ModelLarge: {OK: false, Error: `error, status code: 404, status: 404 Not Found, body: {"error": {"message": "model does not exist"}}`},
					},
					Assistant: openAIModelHealth{OK: true, Error: ""},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "openai enabled, no assistant support",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
				JSONData: json.RawMessage(`{
					"openai": {
						"provider": "openai",
						"url": "%s"
					}
				}`),
			},
			responses: []mockOpenAIHealthResponse{
				{code: http.StatusOK, body: "{}"},
				{code: http.StatusNotFound, body: `{"error": {"message": "model does not exist"}}`},
				{code: http.StatusNotFound, body: `{"error": {"message": "Assistant not available"}}`},
			},
			expDetails: healthCheckDetails{
				OpenAI: openAIHealthDetails{
					Configured: true,
					OK:         true,
					Models: map[Model]openAIModelHealth{
						ModelBase:  {OK: true, Error: ""},
						ModelLarge: {OK: false, Error: `error, status code: 404, status: 404 Not Found, body: {"error": {"message": "model does not exist"}}`},
					},
					Assistant: openAIModelHealth{OK: false, Error: `Assistant not available: error, status code: 404, status: 404 Not Found, body: {"error": {"message": "Assistant not available"}}`},
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
								"url": "%s"
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
				OpenAI: openAIHealthDetails{
					Error:     "No functioning models are available",
					Models:    map[Model]openAIModelHealth{},
					Assistant: openAIModelHealth{OK: false, Error: "Assistant not available"},
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
					"openai": {
						"provider": "openai",
						"url": "%s"
					},
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
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
			},
			responses: []mockOpenAIHealthResponse{
				{code: http.StatusOK, body: "{}"},
				{code: http.StatusNotFound, body: `{"error": {"message": "model does not exist"}}`},
			},
			vService: &mockVectorService{},
			expDetails: healthCheckDetails{
				OpenAI: openAIHealthDetails{
					Configured: true,
					OK:         true,
					Error:      "",
					Models: map[Model]openAIModelHealth{
						ModelBase:  {OK: true, Error: ""},
						ModelLarge: {OK: false, Error: `error, status code: 404, status: 404 Not Found, body: {"error": {"message": "model does not exist"}}`},
					},
					Assistant: openAIModelHealth{OK: false, Error: "Assistant not available: error, status code: 502, status: 502 Bad Gateway, body: "},
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
			server := newMockOpenAIHealthServer(tc.responses)
			defer server.Close()
			tc.settings.JSONData = []byte(fmt.Sprintf(string(tc.settings.JSONData), server.URL))
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
			if details.OpenAI.OK != tc.expDetails.OpenAI.OK ||
				details.OpenAI.Assistant.OK != tc.expDetails.OpenAI.Assistant.OK ||
				details.OpenAI.Assistant.Error != tc.expDetails.OpenAI.Assistant.Error ||
				details.OpenAI.Configured != tc.expDetails.OpenAI.Configured ||
				details.OpenAI.Error != tc.expDetails.OpenAI.Error {
				t.Errorf("OpenAI details should be %+v, got %+v", tc.expDetails.OpenAI, details.OpenAI)
			}
			for k, v := range tc.expDetails.OpenAI.Models {
				if details.OpenAI.Models[k] != v {
					t.Errorf("OpenAI model %s should be %+v, got %+v", k, v, details.OpenAI.Models[k])
				}
			}
			if details.Vector != tc.expDetails.Vector {
				t.Errorf("vector details should be %v, got %v", tc.expDetails.Vector, details.Vector)
			}
		})
	}
}
