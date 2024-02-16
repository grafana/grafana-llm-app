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
	openAIProviderGrafana openAIProvider = "grafana" // via llm-gateway
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

	// apiKey is the user-specified  api key needed to authenticate requests to the OpenAI
	// provider (excluding the LLMGateway). Stored securely.
	apiKey string
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

	EnableGrafanaManagedLLM bool `json:"enableGrafanaManagedLLM"`

	// OpenAI related settings
	OpenAI OpenAISettings `json:"openAI"`

	// VectorDB settings. May rely on OpenAI settings.
	Vector vector.VectorSettings `json:"vector"`

	// LLMGateway provides Grafana-managed OpenAI.
	LLMGateway LLMGatewaySettings `json:"llmGateway"`

	// Pass through settings for plugin data saving
	VectorEmbedderBasicAuthPassword string `json:"vectorEmbedderBasicAuthPassword"`
	VectorStoreBasicAuthPassword    string `json:"vectorStoreBasicAuthPassword"`
}

func loadSettings(appSettings backend.AppInstanceSettings) (*Settings, error) {
	settings := Settings{
		OpenAI: OpenAISettings{
			URL:      "https://api.openai.com",
			Provider: openAIProviderOpenAI,
		},
	}

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

	// Fallback logic if no LLMGateway URL provided by the provisioning/GCom.
	if settings.LLMGateway.URL == "" {
		log.DefaultLogger.Warn("Could not get LLM Gateway URL from config, the LLM Gateway support is disabled")
	}

	switch settings.OpenAI.Provider {
	case openAIProviderOpenAI:
	case openAIProviderAzure:
	case openAIProviderGrafana:
		if settings.LLMGateway.URL == "" {
			// llm-gateway not available, this provider is invalid so switch to disabled
			log.DefaultLogger.Warn("Cannot use LLM Gateway as no URL specified, disabling it")
			settings.OpenAI.Provider = ""
		}
	default:
		// Default to disabled LLM support if an unknown provider was specified.
		log.DefaultLogger.Warn("Unknown OpenAI provider", "provider", settings.OpenAI.Provider)
		settings.OpenAI.Provider = ""
	}

	// Read user's OpenAI key & the LLMGateway key
	settings.OpenAI.apiKey = appSettings.DecryptedSecureJSONData[openAIKey]

	// TenantID and GrafanaCom token are combined as "tenantId:GComToken" and base64 encoded, the following undoes that.
	encodedTenantAndToken, ok := appSettings.DecryptedSecureJSONData[encodedTenantAndTokenKey]
	if ok {
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

	settings.VectorStoreBasicAuthPassword = appSettings.DecryptedSecureJSONData["vectorStoreBasicAuthPassword"]
	settings.VectorEmbedderBasicAuthPassword = appSettings.DecryptedSecureJSONData["vectorEmbedderBasicAuthPassword"]

	return &settings, nil
}
