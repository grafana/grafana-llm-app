package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
)

type mockVectorService struct{}

func (m *mockVectorService) Search(ctx context.Context, collection string, query string, topK uint64, filter map[string]interface{}) ([]store.SearchResult, error) {
	return []store.SearchResult{{Payload: map[string]any{"a": "b"}, Score: 1.0}}, nil
}

func (m *mockVectorService) Health(ctx context.Context) error {
	return nil
}

func (m *mockVectorService) Cancel() {}

type mockProviderHealthResponse struct {
	code int
	body string
}

func newMockProviderHealthServer(responses []mockProviderHealthResponse) *httptest.Server {
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

		responses []mockProviderHealthResponse

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
				LLMProvider: llmProviderHealthDetails{
					Error:  "No functioning models are available",
					Models: map[Model]modelHealth{},
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
				LLMProvider: llmProviderHealthDetails{
					Configured: true,
					Error:      "LLM functionality is disabled",
					Models:     map[Model]modelHealth{},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "vector enabled, no provider",
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
				LLMProvider: llmProviderHealthDetails{
					Error:  "No functioning models are available",
					Models: map[Model]modelHealth{},
				},
				Vector: vectorHealthDetails{
					Enabled: true,
					OK:      true,
				},
				Version: "unknown",
			},
		},
		{
			name: "vector enabled with provider",
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
			responses: []mockProviderHealthResponse{
				{code: http.StatusOK, body: "{}"},
				{code: http.StatusNotFound, body: `{"error": {"message": "model does not exist"}}`},
			},
			vService: &mockVectorService{},
			expDetails: healthCheckDetails{
				LLMProvider: llmProviderHealthDetails{
					Configured: true,
					OK:         true,
					Error:      "",
					Models: map[Model]modelHealth{
						ModelBase:  {OK: true, Error: ""},
						ModelLarge: {OK: false, Error: `error, status code: 404, status: 404 Not Found, message: model does not exist`},
					},
				},
				Vector: vectorHealthDetails{
					Enabled: true,
					OK:      true,
				},
				Version: "unknown",
			},
		},
		{
			name: "anthropic unconfigured shows specific error",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{},
				JSONData: json.RawMessage(`{
					"provider": "anthropic",
					"anthropic": {
						"url": "%s"
					}
				}`),
			},
			expDetails: healthCheckDetails{
				LLMProvider: llmProviderHealthDetails{
					Configured: false,
					OK:         false,
					Error:      "No functioning models are available",
					Models: map[Model]modelHealth{
						ModelBase:  {OK: false, Error: "Anthropic API key is not configured"},
						ModelLarge: {OK: false, Error: "Anthropic API key is not configured"},
					},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "azure unconfigured - no API key and no model mappings",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{},
				JSONData: json.RawMessage(`{
					"provider": "azure",
					"openAI": {
						"url": "%s"
					}
				}`),
			},
			expDetails: healthCheckDetails{
				LLMProvider: llmProviderHealthDetails{
					Configured: false,
					OK:         false,
					Error:      "No functioning models are available",
					Models: map[Model]modelHealth{
						ModelBase:  {OK: false, Error: "Azure OpenAI API key and model mappings are not configured"},
						ModelLarge: {OK: false, Error: "Azure OpenAI API key and model mappings are not configured"},
					},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "azure unconfigured - has API key but no model mappings",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
				JSONData: json.RawMessage(`{
					"provider": "azure",
					"openAI": {
						"url": "%s"
					}
				}`),
			},
			expDetails: healthCheckDetails{
				LLMProvider: llmProviderHealthDetails{
					Configured: false,
					OK:         false,
					Error:      "No functioning models are available",
					Models: map[Model]modelHealth{
						ModelBase:  {OK: false, Error: "Azure model mappings are not configured"},
						ModelLarge: {OK: false, Error: "Azure model mappings are not configured"},
					},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "azure unconfigured - has model mappings but no API key",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{},
				JSONData: json.RawMessage(`{
					"provider": "azure",
					"openAI": {
						"url": "%s",
						"azureModelMapping": [
							["base", "gpt-35-turbo"],
							["large", "gpt-4"]
						]
					}
				}`),
			},
			expDetails: healthCheckDetails{
				LLMProvider: llmProviderHealthDetails{
					Configured: false,
					OK:         false,
					Error:      "No functioning models are available",
					Models: map[Model]modelHealth{
						ModelBase:  {OK: false, Error: "Azure OpenAI API key is not configured"},
						ModelLarge: {OK: false, Error: "Azure OpenAI API key is not configured"},
					},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
		{
			name: "azure configured - has both API key and model mappings",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
				JSONData: json.RawMessage(`{
					"provider": "azure",
					"openAI": {
						"url": "%s",
						"azureModelMapping": [
							["base", "gpt-35-turbo"],
							["large", "gpt-4"]
						]
					}
				}`),
			},
			responses: []mockProviderHealthResponse{
				{code: http.StatusOK, body: "{}"},
				{code: http.StatusOK, body: "{}"},
			},
			expDetails: healthCheckDetails{
				LLMProvider: llmProviderHealthDetails{
					Configured: true,
					OK:         true,
					Error:      "",
					Models: map[Model]modelHealth{
						ModelBase:  {OK: true, Error: ""},
						ModelLarge: {OK: true, Error: ""},
					},
				},
				Vector:  vectorHealthDetails{},
				Version: "unknown",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			server := newMockProviderHealthServer(tc.responses)
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
			if resp == nil { //nolint:staticcheck
				t.Fatal("no response received from CheckHealth")
			}
			var details healthCheckDetails
			if err = json.Unmarshal(resp.JSONDetails, &details); err != nil { //nolint:staticcheck
				t.Errorf("non-JSON response details (%s): %s", resp.JSONDetails, err)
			}

			// Make sure that OpenAI is populated for backwards compatability.
			// We can consider removing this in the future after we are
			// confident frontends have upgraded.
			assert.Equal(t, details.LLMProvider, details.OpenAI)

			if details.LLMProvider.OK != tc.expDetails.LLMProvider.OK ||
				details.LLMProvider.Configured != tc.expDetails.LLMProvider.Configured ||
				details.LLMProvider.Error != tc.expDetails.LLMProvider.Error {
				t.Errorf("LLMProvider details should be %+v, got %+v", tc.expDetails.LLMProvider, details.LLMProvider)
			}
			for k, v := range tc.expDetails.LLMProvider.Models {
				actual := details.LLMProvider.Models[k]
				if actual.OK != v.OK || actual.Error != v.Error {
					t.Errorf("LLMProvider model %s should be %+v, got %+v", k, v, actual)
				}
				if !actual.OK && actual.Error != "" &&
					!strings.Contains(actual.Error, "not configured") &&
					!strings.Contains(actual.Error, "disabled") &&
					actual.Response == nil {
					t.Errorf("LLMProvider model %s API error should have Response field set, got nil", k)
				}
			}
			if details.Vector != tc.expDetails.Vector {
				t.Errorf("vector details should be %v, got %v", tc.expDetails.Vector, details.Vector)
			}
		})
	}
}
