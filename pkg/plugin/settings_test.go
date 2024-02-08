package plugin

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestEmbeddingSettingLogic(t *testing.T) {

	// Set up and run test cases
	for _, tc := range []struct {
		name                   string
		settings               backend.AppInstanceSettings
		embeddingURL           string
		embeddingAuthType      string
		embeddingBasicAuthUser string
	}{
		{
			name: "openai-embedder-option",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"openAI": {
						"url": "https://api.openai.com",
						"provider": "openai"
					},
					"vector": {
						"embed": {
							"type": "openai"
						}
					}
				}`),
				DecryptedSecureJSONData: map[string]string{providerKey: "abcd1234"},
			},
			embeddingURL:           "https://api.openai.com",
			embeddingAuthType:      "openai-key-auth",
			embeddingBasicAuthUser: "",
		},
		{
			name: "grafana-vector-api-no-auth",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"vector": {
						"embed": {
							"type": "grafana/vectorapi",
							"grafanaVectorAPI": {
								"url": "https://api.example.com",
								"authType": "no-auth"
							}
						}
					}
				}`),
			},
			embeddingURL:           "https://api.example.com",
			embeddingAuthType:      "no-auth",
			embeddingBasicAuthUser: "",
		},
		{
			name: "grafana-vector-api-basic-auth",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"vector": {
						"embed": {
							"type": "grafana/vectorapi",
							"grafanaVectorAPI": {
								"url": "https://api.example.com",
								"authType": "basic-auth",
								"basicAuthUser": "test"
							}
						}
					}
				}`),
			},
			embeddingURL:           "https://api.example.com",
			embeddingAuthType:      "basic-auth",
			embeddingBasicAuthUser: "test",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := loadSettings(tc.settings)
			if err != nil {
				t.Errorf("loadSettings failed: %s", err)
			}

			// Assert that the settings are loaded correctly
			if settings.Vector.Embed.Type == "openai" {
				if settings.Vector.Embed.OpenAI.URL != tc.embeddingURL {
					t.Errorf("expected embedding URL to be %s, got %s", tc.embeddingURL, settings.Vector.Embed.OpenAI.URL)
				}

				if settings.Vector.Embed.OpenAI.AuthType != tc.embeddingAuthType {
					t.Errorf("expected embedding auth type to be %s, got %s", tc.embeddingAuthType, settings.Vector.Embed.OpenAI.AuthType)
				}
			} else if settings.Vector.Embed.Type == "grafana/vectorapi" {
				if settings.Vector.Embed.GrafanaVectorAPISettings.URL != tc.embeddingURL {
					t.Errorf("expected embedding URL to be %s, got %s", tc.embeddingURL, settings.Vector.Embed.GrafanaVectorAPISettings.URL)
				}

				if settings.Vector.Embed.GrafanaVectorAPISettings.AuthType != tc.embeddingAuthType {
					t.Errorf("expected embedding auth type to be %s, got %s", tc.embeddingAuthType, settings.Vector.Embed.GrafanaVectorAPISettings.AuthType)
				}

				if settings.Vector.Embed.GrafanaVectorAPISettings.BasicAuthUser != tc.embeddingBasicAuthUser {
					t.Errorf("expected embedding basic auth user to be %s, got %s", tc.embeddingBasicAuthUser, settings.Vector.Embed.GrafanaVectorAPISettings.BasicAuthUser)
				}
			} else {
				t.Errorf("expected embedding type to be openai or grafana/vectorapi, got %s", settings.Vector.Embed.Type)
			}
		})
	}
}

func TestManagedLLMSettingsLogic(t *testing.T) {

	// Set up and run test cases
	for _, tc := range []struct {
		name          string
		settings      backend.AppInstanceSettings
		llmGatewayURL string
		llmIsOptIn    bool
	}{
		{
			name: "grafana-llm-gateway-no-explicit-opt-in",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"llmGateway": {
						"url": "https://llm-gateway-prod-api-eu-west.grafana.net"
					}
				}`),
			},
			llmGatewayURL: "https://llm-gateway-prod-api-eu-west.grafana.net",
			llmIsOptIn:    false,
		},
		{
			name: "grafana-llm-gateway-explicit-opt-in",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"llmGateway": {
						"url": "https://llm-gateway-prod-api-eu-west.grafana.net",
						"isOptIn": true
					}
				}`),
			},
			llmGatewayURL: "https://llm-gateway-prod-api-eu-west.grafana.net",
			llmIsOptIn:    true,
		},
		{
			name: "grafana-llm-gateway-explicit-opt-out",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"llmGateway": {
						"url": "https://llm-gateway-prod-api-eu-west.grafana.net",
						"isOptIn": false
					}
				}`),
			},
			llmGatewayURL: "https://llm-gateway-prod-api-eu-west.grafana.net",
			llmIsOptIn:    false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := loadSettings(tc.settings)
			if err != nil {
				t.Errorf("loadSettings failed: %s", err)
			}

			// Assert that the settings are loaded correctly
			if settings.LLMGateway.URL != tc.llmGatewayURL {
				t.Errorf("expected llm gateway URL to be %s, got %s", tc.llmGatewayURL, settings.LLMGateway.URL)
			}

			if settings.LLMGateway.IsOptIn != tc.llmIsOptIn {
				t.Errorf("expected llm opt in status to be %t, got %t", tc.llmIsOptIn, settings.LLMGateway.IsOptIn)
			}
		})
	}
}
