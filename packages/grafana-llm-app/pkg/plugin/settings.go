package plugin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/embed"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const openAIKey = "openAIKey"
const encodedTenantAndTokenKey = "base64EncodedAccessToken"

type openAIProvider string

const (
	openAIProviderOpenAI  openAIProvider = "openai"
	openAIProviderAzure   openAIProvider = "azure"
	openAIProviderCustom  openAIProvider = "custom"
	openAIProviderGrafana openAIProvider = "grafana" // via llm-gateway
	openAIProviderTest    openAIProvider = "test"
)

// OpenAISettings contains the user-specified OpenAI connection details
type OpenAISettings struct {
	// The URL to the OpenAI provider
	URL string `json:"url"`

	// The OrgID to be passed to OpenAI in requests
	OrganizationID string `json:"organizationId"`

	// What OpenAI provider the user selected. Note this can specify using the LLMGateway
	Provider openAIProvider `json:"provider"`

	// Model mappings required for Azure's OpenAI
	AzureMapping [][]string `json:"azureModelMapping"`

	// Disabled marks if a user has explicitly disabled LLM functionality.
	Disabled bool `json:"disabled"`

	// apiKey is the user-specified  api key needed to authenticate requests to the OpenAI
	// provider (excluding the LLMGateway). Stored securely.
	apiKey string

	// TestProvider contains the settings for the test provider.
	// Only used when Provider is openAIProviderTest.
	TestProvider testProvider `json:"testProvider,omitempty"`
}

func (s OpenAISettings) Configured() bool {
	// If disabled has been selected than the plugin has been configured.
	if s.Disabled {
		return true
	}

	switch s.Provider {
	case openAIProviderGrafana, openAIProviderCustom, openAIProviderTest:
		return true
	case openAIProviderAzure:
		// Require some mappings for use with Azure.
		if len(s.AzureMapping) == 0 {
			return false
		}
		// Still need to check the same conditions as openAIProviderOpenAI.
		fallthrough
	case openAIProviderOpenAI:
		return s.apiKey != ""
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

var DEFAULT_MODEL_SETTINGS = &ModelSettings{
	Default: ModelBase,
	Mapping: map[Model]string{
		ModelBase:  "gpt-4o-mini",
		ModelLarge: "gpt-4o",
	},
}

// LLMGatewaySettings contains the configuration for the Grafana Managed Key LLM solution.
type LLMGatewaySettings struct {
	// This is the URL of the LLM endpoint of the machine learning backend which proxies
	// the request to our llm-gateway. If empty, the gateway is disabled.
	URL string `json:"url"`
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

	// OpenAI related settings
	OpenAI OpenAISettings `json:"openAI"`

	// VectorDB settings. May rely on OpenAI settings.
	Vector vector.VectorSettings `json:"vector"`

	// Models contains the user-specified models.
	Models *ModelSettings `json:"models"`

	// LLMGateway provides Grafana-managed OpenAI.
	LLMGateway LLMGatewaySettings `json:"llmGateway"`
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

	// We need to handle the case where the user has customized the URL,
	// then reverted that customization so that the JSON data includes
	// an empty string.
	if settings.OpenAI.URL == "" {
		settings.OpenAI.URL = "https://api.openai.com"
	}
	if settings.Vector.Embed.Type == embed.EmbedderOpenAI {
		settings.Vector.Embed.OpenAI.URL = settings.OpenAI.URL
		settings.Vector.Embed.OpenAI.AuthType = "openai-key-auth"
	}
	const openAIKey = "openAIKey"
	const encodedTenantAndTokenKey = "base64EncodedAccessToken"
	// Fallback logic if no LLMGateway URL provided by the provisioning/GCom.
	if settings.LLMGateway.URL == "" {
		log.DefaultLogger.Warn("Could not get LLM Gateway URL from config, the LLM Gateway support is disabled")
	}

	switch settings.OpenAI.Provider {
	case openAIProviderOpenAI:
	case openAIProviderAzure:
	case openAIProviderCustom:
	case openAIProviderGrafana:
		if settings.LLMGateway.URL == "" {
			// llm-gateway not available, this provider is invalid so switch to disabled
			log.DefaultLogger.Warn("Cannot use LLM Gateway as no URL specified, disabling it")
			settings.OpenAI.Provider = ""
		}
	case openAIProviderTest:
		settings.OpenAI.Provider = openAIProviderTest
	default:
		// Default to disabled LLM support if an unknown provider was specified.
		log.DefaultLogger.Warn("Unknown OpenAI provider", "provider", settings.OpenAI.Provider)
		settings.OpenAI.Provider = ""
	}

	settings.DecryptedSecureJSONData = appSettings.DecryptedSecureJSONData

	settings.OpenAI.apiKey = settings.DecryptedSecureJSONData[openAIKey]

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
