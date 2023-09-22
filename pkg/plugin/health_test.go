package plugin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// TestCheckHealth tests CheckHealth calls, using backend.CheckHealthRequest and backend.CheckHealthResponse.
func TestCheckHealth(t *testing.T) {

	// Set up and run test cases
	for _, tc := range []struct {
		name     string
		settings backend.AppInstanceSettings

		expDetails healthCheckResponse
	}{
		{
			name: "everything disabled",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{},
			},
			expDetails: healthCheckResponse{
				OpenAIEnabled: false,
				VectorEnabled: false,
				Version:       "unknown",
			},
		},
		{
			name: "openai enabled",
			settings: backend.AppInstanceSettings{
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
			},
			expDetails: healthCheckResponse{
				OpenAIEnabled: true,
				VectorEnabled: false,
				Version:       "unknown",
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
			expDetails: healthCheckResponse{
				OpenAIEnabled: false,
				VectorEnabled: true,
				Version:       "unknown",
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
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
			},
			expDetails: healthCheckResponse{
				OpenAIEnabled: true,
				VectorEnabled: true,
				Version:       "unknown",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize app
			inst, err := NewApp(tc.settings)
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
			// Request by calling CheckHealth.
			resp, err := app.CheckHealth(context.Background(), &backend.CheckHealthRequest{
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
			var details healthCheckResponse
			if err = json.Unmarshal(resp.JSONDetails, &details); err != nil {
				t.Errorf("non-JSON response details (%s): %s", resp.JSONDetails, err)
			}
			if details != tc.expDetails {
				t.Errorf("response details should be %v, got %v", tc.expDetails, details)
			}
		})
	}
}
