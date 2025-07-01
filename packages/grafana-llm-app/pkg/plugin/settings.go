package plugin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/embed"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

const (
	openAIKey                = "openAIKey"
	encodedTenantAndTokenKey = "base64EncodedAccessToken"
)

var (
	// This has to be a var so we can take its address later.
	defaultOpenAIAPIPath = "/v1"
)

type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAzure     ProviderType = "azure"
	ProviderTypeCustom    ProviderType = "custom"
	ProviderTypeGrafana   ProviderType = "grafana" // via llm-gateway
	ProviderTypeTest      ProviderType = "test"
	ProviderTypeAnthropic ProviderType = "anthropic"
)

// OpenAISettings contains the user-specified OpenAI connection details
type OpenAISettings struct {
	// The URL to the OpenAI provider
	URL string `json:"url"`

	// The API path to append to the URL.
	// If nil, the default path of /v1 is used.
	APIPath *string `json:"apiPath"`

	// The OrgID to be passed to OpenAI in requests
	OrganizationID string `json:"organizationId"`

	// What OpenAI provider the user selected. Note this can specify using the LLMGateway
	// Deprecated: Use Settings.Provider instead
	Provider ProviderType `json:"provider"`

	// Model mappings required for Azure's OpenAI
	AzureMapping [][]string `json:"azureModelMapping"`

	// Disabled marks if a user has explicitly disabled LLM functionality.
	// Deprecated: Use Settings.Disabled instead
	Disabled bool `json:"disabled"`

	// apiKey is the user-specified  api key needed to authenticate requests to the OpenAI
	// provider (excluding the LLMGateway). Stored securely.
	apiKey string

	// TestProvider contains the settings for the test provider.
	// Only used when Provider is ProviderTypeTest.
	TestProvider testProvider `json:"testProvider,omitempty"`
}

// AnthropicSettings contains Anthropic-specific settings
type AnthropicSettings struct {
	// The URL to the provider's API
	URL string `json:"url"`

	// apiKey is the provider-specific API key needed to authenticate requests
	// Stored securely.
	apiKey string
}

// Configured returns whether the provider has been configured
func (s *Settings) Configured() bool {
	// If disabled has been selected than the provider has been configured.
	if s.Disabled || s.OpenAI.Disabled { // Check both for backward compatibility
		return true
	}

	// For backwards compatibility, check OpenAI.Provider if Settings.Provider is empty
	provider := s.Provider
	if provider == "" {
		provider = s.OpenAI.Provider
	}

	switch provider {
	case ProviderTypeGrafana, ProviderTypeCustom, ProviderTypeTest:
		return true
	case ProviderTypeAzure:
		// Require some mappings for use with Azure.
		if len(s.OpenAI.AzureMapping) == 0 {
			return false
		}
		// Still need to check the same conditions as openAIProviderOpenAI.
		fallthrough
	case ProviderTypeOpenAI:
		return s.OpenAI.apiKey != ""
	case ProviderTypeAnthropic:
		return s.Anthropic.apiKey != ""
	}
	// Unknown or empty provider means configuration needs to be updated.
	return false
}

type ModelMapping struct {
	Model Model  `json:"model"`
	Name  string `json:"name"`
}

type ModelSettings struct {
	// Default model to use when no model is defined, or the model is not found.
	Default Model `json:"default"`

	// Mapping is mapping from our abstract model names to the provider's model names.
	Mapping map[Model]string `json:"mapping"`
}

func (c ModelSettings) getModel(model Model) string {
	// Helper function to get the name of a model.
	if name, ok := c.Mapping[model]; ok {
		return name
	}
	// If the model is not found, return the default model.
	return c.getModel(c.Default)
}

func defaultModelSettings(provider ProviderType) *ModelSettings {
	switch provider {
	case ProviderTypeAnthropic:
		return &ModelSettings{
			Default: ModelBase,
			Mapping: map[Model]string{
				ModelBase:  string(anthropic.ModelClaude4Sonnet20250514),
				ModelLarge: string(anthropic.ModelClaude4Sonnet20250514),
			},
		}
	default:
		return &ModelSettings{
			Default: ModelBase,
			Mapping: map[Model]string{
				ModelBase:  openai.GPT4Dot1Mini,
				ModelLarge: openai.GPT4Dot1,
			},
		}
	}
}

// LLMGatewaySettings contains the configuration for the Grafana Managed Key LLM solution.
type LLMGatewaySettings struct {
	// This is the URL of the LLM endpoint of the machine learning backend which proxies
	// the request to our llm-gateway. If empty, the gateway is disabled.
	URL string `json:"url"`
}

// MCPSettings contains the configuration for the Grafana MCP server.
type MCPSettings struct {
	// Disabled indicates whether the MCP server should be disabled.
	Disabled bool `json:"disabled"`
}

// Settings contains the plugin's settings and secrets required by the plugin backend.
type Settings struct {
	// Tenant is the stack ID (Hosted Grafana ID) of the instance this plugin
	// is running on.
	Tenant string

	// GrafanaComAPIKey is a grafana.com Editor API key used to interact with the grafana.com API.
	//
	// It is created by the grafana.com API when the plugin is first provisioned for a tenant.
	//
	// It is used when persisting the plugin's settings after setup.
	GrafanaComAPIKey string

	DecryptedSecureJSONData map[string]string

	EnableGrafanaManagedLLM bool `json:"enableGrafanaManagedLLM"`

	// Provider type indicates which provider implementation to use
	Provider ProviderType `json:"provider"`

	// Disabled marks if a user has explicitly disabled LLM functionality.
	Disabled bool `json:"disabled"`

	// OpenAI related settings
	OpenAI OpenAISettings `json:"openAI"`

	// Anthropic related settings
	Anthropic AnthropicSettings `json:"anthropic"`

	// VectorDB settings. May rely on OpenAI settings.
	Vector vector.VectorSettings `json:"vector"`

	// Models contains the user-specified models.
	Models *ModelSettings `json:"models"`

	// LLMGateway provides Grafana-managed OpenAI.
	LLMGateway LLMGatewaySettings `json:"llmGateway"`

	// Allows enabling the dev sandbox on the plugin page.
	EnableDevSandbox bool `json:"enableDevSandbox"`

	// MCP settings.
	MCP MCPSettings `json:"mcp"`
}

func loadSettings(appSettings backend.AppInstanceSettings) (*Settings, error) {
	settings := Settings{OpenAI: OpenAISettings{TestProvider: defaultTestProvider()}}

	if len(appSettings.JSONData) != 0 {
		err := json.Unmarshal(appSettings.JSONData, &settings)
		if err != nil {
			log.DefaultLogger.Error(err.Error())
			return nil, err
		}
	}

	// Handle migration of Disabled field from OpenAI to top level
	if !settings.Disabled && settings.OpenAI.Disabled {
		settings.Disabled = settings.OpenAI.Disabled
	}

	// We need to handle the case where the user has customized the URL,
	// then reverted that customization so that the JSON data includes
	// an empty string.
	if settings.OpenAI.URL == "" {
		settings.OpenAI.URL = "https://api.openai.com"
	}
	// Use default API path if not overridden in settings.
	if settings.OpenAI.APIPath == nil {
		settings.OpenAI.APIPath = &defaultOpenAIAPIPath
	}
	if settings.Anthropic.URL == "" {
		settings.Anthropic.URL = "https://api.anthropic.com"
	}
	if settings.Vector.Embed.Type == embed.EmbedderOpenAI {
		settings.Vector.Embed.OpenAI.URL = settings.OpenAI.URL
		settings.Vector.Embed.OpenAI.AuthType = "openai-key-auth"
	}

	provider := settings.getEffectiveProvider()

	// Verify this is a known provider type
	knownProvider := provider == ProviderTypeOpenAI ||
		provider == ProviderTypeAzure ||
		provider == ProviderTypeCustom ||
		provider == ProviderTypeGrafana ||
		provider == ProviderTypeTest ||
		provider == ProviderTypeAnthropic

	if !knownProvider {
		log.DefaultLogger.Warn("Unknown provider", "provider", settings.Provider)
		settings.OpenAI.Provider = ""
		settings.Provider = ""
	}

	if provider == ProviderTypeGrafana && settings.LLMGateway.URL == "" {
		log.DefaultLogger.Warn("Cannot use LLM Gateway as no URL specified, disabling it")
		settings.OpenAI.Provider = ""
		settings.Provider = ""
	}

	settings.DecryptedSecureJSONData = appSettings.DecryptedSecureJSONData

	settings.OpenAI.apiKey = settings.DecryptedSecureJSONData[openAIKey]
	settings.Anthropic.apiKey = settings.DecryptedSecureJSONData["anthropicKey"]

	// TenantID and GrafanaCom token are combined as "tenantId:GComToken" and base64 encoded, the following undoes that.
	encodedTenantAndToken := settings.DecryptedSecureJSONData[encodedTenantAndTokenKey]
	if encodedTenantAndToken != "" {
		token, err := base64.StdEncoding.DecodeString(encodedTenantAndToken)
		if err != nil {
			log.DefaultLogger.Error(err.Error())
			return nil, err
		}
		tokenParts := strings.Split(strings.TrimSpace(string(token)), ":")
		if len(tokenParts) != 2 {
			return nil, errors.New("invalid access token")
		}
		settings.Tenant = strings.TrimSpace(tokenParts[0])
		if settings.Tenant == "" {
			return nil, errors.New("invalid tenant")
		}
		settings.GrafanaComAPIKey = strings.TrimSpace(tokenParts[1])
		if settings.GrafanaComAPIKey == "" {
			return nil, errors.New("invalid grafana.com API key")
		}
	}

	return &settings, nil
}

// getEffectiveProvider returns the effective provider type, handling backward compatibility
// where Provider was previously stored in OpenAI.Provider
func (s *Settings) getEffectiveProvider() ProviderType {
	if s.Provider == "" {
		return s.OpenAI.Provider
	}
	return s.Provider
}
