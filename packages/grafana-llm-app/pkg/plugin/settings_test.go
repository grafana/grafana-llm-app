package plugin

import (
	"testing"

	"github.com/grafana/grafana-llm-app/pkg/mcp"
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
			switch settings.Vector.Embed.Type {
			case "openai":
				if settings.Vector.Embed.OpenAI.URL != tc.embeddingURL {
					t.Errorf("expected embedding URL to be %s, got %s", tc.embeddingURL, settings.Vector.Embed.OpenAI.URL)
				}

				if settings.Vector.Embed.OpenAI.AuthType != tc.embeddingAuthType {
					t.Errorf("expected embedding auth type to be %s, got %s", tc.embeddingAuthType, settings.Vector.Embed.OpenAI.AuthType)
				}
			case "grafana/vectorapi":
				if settings.Vector.Embed.GrafanaVectorAPISettings.URL != tc.embeddingURL {
					t.Errorf("expected embedding URL to be %s, got %s", tc.embeddingURL, settings.Vector.Embed.GrafanaVectorAPISettings.URL)
				}

				if settings.Vector.Embed.GrafanaVectorAPISettings.AuthType != tc.embeddingAuthType {
					t.Errorf("expected embedding auth type to be %s, got %s", tc.embeddingAuthType, settings.Vector.Embed.GrafanaVectorAPISettings.AuthType)
				}

				if settings.Vector.Embed.GrafanaVectorAPISettings.BasicAuthUser != tc.embeddingBasicAuthUser {
					t.Errorf("expected embedding basic auth user to be %s, got %s", tc.embeddingBasicAuthUser, settings.Vector.Embed.GrafanaVectorAPISettings.BasicAuthUser)
				}
			default:
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
				Disabled: true,
			},
			configured: true,
		},
		{
			testName: "disabled with otherwise valid configuration",
			settings: Settings{
				Provider: ProviderTypeGrafana,
				Disabled: true,
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

func TestDisabledBackwardCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name            string
		settings        backend.AppInstanceSettings
		expectedResult  bool
		expectedMessage string
	}{
		{
			name: "neither disabled flag set",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{}`),
			},
			expectedResult: false,
		},
		{
			name: "root disabled flag set to true",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"disabled": true}`),
			},
			expectedResult: true,
		},
		{
			name: "openai disabled flag set to true (legacy)",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"openAI": {"disabled": true}}`),
			},
			expectedResult: true,
		},
		{
			name: "both flags set to true",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"disabled": true, "openAI": {"disabled": true}}`),
			},
			expectedResult: true,
		},
		{
			name: "root disabled true, openai disabled false",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"disabled": true, "openAI": {"disabled": false}}`),
			},
			expectedResult: true,
		},
		{
			name: "root disabled false, openai disabled true",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"disabled": false, "openAI": {"disabled": true}}`),
			},
			expectedResult: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := loadSettings(tc.settings)
			if err != nil {
				t.Errorf("loadSettings failed: %s", err)
			}

			if settings.Disabled != tc.expectedResult {
				t.Errorf("expected Disabled to be %v, got %v", tc.expectedResult, settings.Disabled)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }

func TestMCPToolsetsIsEnabled(t *testing.T) {
	for _, tc := range []struct {
		name     string
		toolsets MCPToolsets
		toolset  mcp.Toolset
		expected bool
	}{
		{
			name:     "nil pointer defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetSearch,
			expected: true,
		},
		{
			name:     "explicitly enabled",
			toolsets: MCPToolsets{Search: boolPtr(true)},
			toolset:  mcp.ToolsetSearch,
			expected: true,
		},
		{
			name:     "explicitly disabled",
			toolsets: MCPToolsets{Search: boolPtr(false)},
			toolset:  mcp.ToolsetSearch,
			expected: false,
		},
		{
			name:     "unknown toolset returns false",
			toolsets: MCPToolsets{},
			toolset:  "unknown",
			expected: false,
		},
		{
			name:     "datasource nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetDatasource,
			expected: true,
		},
		{
			name:     "incident disabled",
			toolsets: MCPToolsets{Incident: boolPtr(false)},
			toolset:  mcp.ToolsetIncident,
			expected: false,
		},
		{
			name:     "prometheus nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetPrometheus,
			expected: true,
		},
		{
			name:     "loki disabled",
			toolsets: MCPToolsets{Loki: boolPtr(false)},
			toolset:  mcp.ToolsetLoki,
			expected: false,
		},
		{
			name:     "alerting enabled",
			toolsets: MCPToolsets{Alerting: boolPtr(true)},
			toolset:  mcp.ToolsetAlerting,
			expected: true,
		},
		{
			name:     "dashboard disabled",
			toolsets: MCPToolsets{Dashboard: boolPtr(false)},
			toolset:  mcp.ToolsetDashboard,
			expected: false,
		},
		{
			name:     "oncall nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetOnCall,
			expected: true,
		},
		{
			name:     "asserts disabled",
			toolsets: MCPToolsets{Asserts: boolPtr(false)},
			toolset:  mcp.ToolsetAsserts,
			expected: false,
		},
		{
			name:     "sift disabled",
			toolsets: MCPToolsets{Sift: boolPtr(false)},
			toolset:  mcp.ToolsetSift,
			expected: false,
		},
		{
			name:     "pyroscope nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetPyroscope,
			expected: true,
		},
		{
			name:     "pyroscope disabled",
			toolsets: MCPToolsets{Pyroscope: boolPtr(false)},
			toolset:  mcp.ToolsetPyroscope,
			expected: false,
		},
		{
			name:     "navigation nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetNavigation,
			expected: true,
		},
		{
			name:     "navigation disabled",
			toolsets: MCPToolsets{Navigation: boolPtr(false)},
			toolset:  mcp.ToolsetNavigation,
			expected: false,
		},
		{
			name:     "annotations nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetAnnotations,
			expected: true,
		},
		{
			name:     "annotations disabled",
			toolsets: MCPToolsets{Annotations: boolPtr(false)},
			toolset:  mcp.ToolsetAnnotations,
			expected: false,
		},
		{
			name:     "rendering nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetRendering,
			expected: true,
		},
		{
			name:     "rendering disabled",
			toolsets: MCPToolsets{Rendering: boolPtr(false)},
			toolset:  mcp.ToolsetRendering,
			expected: false,
		},
		{
			name:     "admin nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetAdmin,
			expected: true,
		},
		{
			name:     "admin disabled",
			toolsets: MCPToolsets{Admin: boolPtr(false)},
			toolset:  mcp.ToolsetAdmin,
			expected: false,
		},
		{
			name:     "clickhouse nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetClickHouse,
			expected: true,
		},
		{
			name:     "clickhouse disabled",
			toolsets: MCPToolsets{ClickHouse: boolPtr(false)},
			toolset:  mcp.ToolsetClickHouse,
			expected: false,
		},
		{
			name:     "cloudwatch nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetCloudWatch,
			expected: true,
		},
		{
			name:     "cloudwatch disabled",
			toolsets: MCPToolsets{CloudWatch: boolPtr(false)},
			toolset:  mcp.ToolsetCloudWatch,
			expected: false,
		},
		{
			name:     "elasticsearch nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetElasticsearch,
			expected: true,
		},
		{
			name:     "elasticsearch disabled",
			toolsets: MCPToolsets{Elasticsearch: boolPtr(false)},
			toolset:  mcp.ToolsetElasticsearch,
			expected: false,
		},
		{
			name:     "examples nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetExamples,
			expected: true,
		},
		{
			name:     "examples disabled",
			toolsets: MCPToolsets{Examples: boolPtr(false)},
			toolset:  mcp.ToolsetExamples,
			expected: false,
		},
		{
			name:     "searchlogs nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetSearchLogs,
			expected: true,
		},
		{
			name:     "searchlogs disabled",
			toolsets: MCPToolsets{SearchLogs: boolPtr(false)},
			toolset:  mcp.ToolsetSearchLogs,
			expected: false,
		},
		{
			name:     "folder nil defaults to enabled",
			toolsets: MCPToolsets{},
			toolset:  mcp.ToolsetFolder,
			expected: true,
		},
		{
			name:     "folder disabled",
			toolsets: MCPToolsets{Folder: boolPtr(false)},
			toolset:  mcp.ToolsetFolder,
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.toolsets.IsEnabled(tc.toolset)
			if result != tc.expected {
				t.Errorf("IsEnabled(%q) = %v, want %v", tc.toolset, result, tc.expected)
			}
		})
	}
}

func TestMCPToolsetsSettings(t *testing.T) {
	for _, tc := range []struct {
		name            string
		settings        backend.AppInstanceSettings
		enabledToolset  mcp.Toolset
		expectedEnabled bool
	}{
		{
			name: "no toolsets config means all enabled",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"mcp": {}}`),
			},
			enabledToolset:  mcp.ToolsetSearch,
			expectedEnabled: true,
		},
		{
			name: "empty toolsets means all enabled",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"mcp": {"toolsets": {}}}`),
			},
			enabledToolset:  mcp.ToolsetIncident,
			expectedEnabled: true,
		},
		{
			name: "incident explicitly disabled",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"mcp": {"toolsets": {"incident": false}}}`),
			},
			enabledToolset:  mcp.ToolsetIncident,
			expectedEnabled: false,
		},
		{
			name: "oncall explicitly disabled, search still enabled",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"mcp": {"toolsets": {"oncall": false}}}`),
			},
			enabledToolset:  mcp.ToolsetSearch,
			expectedEnabled: true,
		},
		{
			name: "oncall explicitly disabled",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"mcp": {"toolsets": {"oncall": false}}}`),
			},
			enabledToolset:  mcp.ToolsetOnCall,
			expectedEnabled: false,
		},
		{
			name: "prometheus explicitly enabled",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{"mcp": {"toolsets": {"prometheus": true}}}`),
			},
			enabledToolset:  mcp.ToolsetPrometheus,
			expectedEnabled: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := loadSettings(tc.settings)
			if err != nil {
				t.Fatalf("loadSettings failed: %s", err)
			}
			result := settings.MCP.Toolsets.IsEnabled(tc.enabledToolset)
			if result != tc.expectedEnabled {
				t.Errorf("IsEnabled(%q) = %v, want %v", tc.enabledToolset, result, tc.expectedEnabled)
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
