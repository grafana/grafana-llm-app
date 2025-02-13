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
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
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

		})
	}
}

func TestConfigured(t *testing.T) {
	for _, tc := range []struct {
		testName   string
		settings   Settings
		configured bool
	}{
		{
			testName:   "empty",
			settings:   Settings{},
			configured: false,
		},
		{
			testName: "disabled",
			settings: Settings{
				OpenAI: OpenAISettings{
					Disabled: true,
				},
			},
			configured: true,
		},
		{
			testName: "disabled with otherwise valid configuration",
			settings: Settings{
				Provider: ProviderTypeGrafana,
				OpenAI: OpenAISettings{
					Disabled: true,
				},
			},
			configured: true,
		},
		// OpenAI tests with root provider
		{
			testName: "openai without api key (root provider)",
			settings: Settings{
				Provider: ProviderTypeOpenAI,
			},
			configured: false,
		},
		{
			testName: "openai with api key (root provider)",
			settings: Settings{
				Provider: ProviderTypeOpenAI,
				OpenAI: OpenAISettings{
					apiKey: "hello",
				},
			},
			configured: true,
		},
		// OpenAI tests with legacy provider
		{
			testName: "openai without api key (legacy provider)",
			settings: Settings{
				OpenAI: OpenAISettings{
					Provider: ProviderTypeOpenAI,
				},
			},
			configured: false,
		},
		{
			testName: "openai with api key (legacy provider)",
			settings: Settings{
				OpenAI: OpenAISettings{
					Provider: ProviderTypeOpenAI,
					apiKey:   "hello",
				},
			},
			configured: true,
		},
		// Azure tests with root provider
		{
			testName: "azure without mapping (root provider)",
			settings: Settings{
				Provider: ProviderTypeAzure,
				OpenAI: OpenAISettings{
					apiKey: "hello",
				},
			},
			configured: false,
		},
		{
			testName: "azure with mapping without api key (root provider)",
			settings: Settings{
				Provider: ProviderTypeAzure,
				OpenAI: OpenAISettings{
					AzureMapping: [][]string{
						{ModelBase, "azuredeployment"},
						{ModelLarge, "largeazuredeployment"},
					},
				},
			},
			configured: false,
		},
		{
			testName: "azure valid (root provider)",
			settings: Settings{
				Provider: ProviderTypeAzure,
				OpenAI: OpenAISettings{
					apiKey: "hello",
					AzureMapping: [][]string{
						{ModelBase, "azuredeployment"},
						{ModelLarge, "largeazuredeployment"},
					},
				},
			},
			configured: true,
		},
		// Azure tests with legacy provider
		{
			testName: "azure without mapping (legacy provider)",
			settings: Settings{
				OpenAI: OpenAISettings{
					Provider: ProviderTypeAzure,
					apiKey:   "hello",
				},
			},
			configured: false,
		},
		{
			testName: "azure with mapping without api key (legacy provider)",
			settings: Settings{
				OpenAI: OpenAISettings{
					Provider: ProviderTypeAzure,
					AzureMapping: [][]string{
						{ModelBase, "azuredeployment"},
						{ModelLarge, "largeazuredeployment"},
					},
				},
			},
			configured: false,
		},
		{
			testName: "azure valid (legacy provider)",
			settings: Settings{
				OpenAI: OpenAISettings{
					Provider: ProviderTypeAzure,
					apiKey:   "hello",
					AzureMapping: [][]string{
						{ModelBase, "azuredeployment"},
						{ModelLarge, "largeazuredeployment"},
					},
				},
			},
			configured: true,
		},
		// Grafana tests with root provider
		{
			testName: "grafana provider (root)",
			settings: Settings{
				Provider: ProviderTypeGrafana,
			},
			configured: true,
		},
		// Grafana tests with legacy provider
		{
			testName: "grafana provider (legacy)",
			settings: Settings{
				OpenAI: OpenAISettings{
					Provider: ProviderTypeGrafana,
				},
			},
			configured: true,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			if tc.configured != tc.settings.Configured() {
				t.Errorf("expected configured to be `%t`", tc.configured)
			}
		})
	}
}

func TestGetEffectiveProvider(t *testing.T) {
	for _, tc := range []struct {
		name             string
		settings         Settings
		expectedProvider ProviderType
	}{
		{
			name: "both providers empty",
			settings: Settings{
				Provider: "",
				OpenAI: OpenAISettings{
					Provider: "",
				},
			},
			expectedProvider: "",
		},
		{
			name: "only root provider set",
			settings: Settings{
				Provider: ProviderTypeGrafana,
				OpenAI: OpenAISettings{
					Provider: "",
				},
			},
			expectedProvider: ProviderTypeGrafana,
		},
		{
			name: "only openai provider set (backward compatibility)",
			settings: Settings{
				Provider: "",
				OpenAI: OpenAISettings{
					Provider: ProviderTypeOpenAI,
				},
			},
			expectedProvider: ProviderTypeOpenAI,
		},
		{
			name: "both providers set (root provider takes precedence)",
			settings: Settings{
				Provider: ProviderTypeGrafana,
				OpenAI: OpenAISettings{
					Provider: ProviderTypeOpenAI,
				},
			},
			expectedProvider: ProviderTypeGrafana,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := tc.settings.getEffectiveProvider()
			if provider != tc.expectedProvider {
				t.Errorf("expected provider to be %s, got %s", tc.expectedProvider, provider)
			}
		})
	}
}
