package plugin

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/embed"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const openAIKey = "openAIKey"

type openAIProvider string

const (
	openAIProviderOpenAI  openAIProvider = "openai"
	openAIProviderAzure   openAIProvider = "azure"
	openAIProviderGrafana openAIProvider = "grafana" // via llm-gateway
)

type OpenAISettings struct {
	URL            string         `json:"url"`
	OrganizationID string         `json:"organizationId"`
	Provider       openAIProvider `json:"provider"`
	AzureMapping   [][]string     `json:"azureModelMapping"`
	apiKey         string
}

// LLMGatewaySettings contains the cnfiguration for the  Grafana Managed Key LLM solution.
type LLMGatewaySettings struct {
	// This is the URL of the LLM endpoint of the machine learning backend which proxies
	// the request to our llm-gateway.
	URL string `json:"url"`

	//apiKey is the api key needed to authenticate requests to the LLM gateway
	APIKey string `json:"apiKey"`

	// optInStatus indicates if customer has enabled the Grafana Managed Key LLM.
	// If not specified, this is unmarshalled to false.
	OptInStatus bool `json:"optInStatus"`
}

type Settings struct {
	OpenAI OpenAISettings `json:"openAI"`

	Vector vector.VectorSettings `json:"vector"`

	LLMGateway LLMGatewaySettings `json:"llmGateway"`
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

	switch settings.OpenAI.Provider {
	case openAIProviderOpenAI:
	case openAIProviderAzure:
	case openAIProviderGrafana:
		if settings.LLMGateway.URL == "" {
			// Attempt to get the LLM Gateway URL from the LLM_GATEWAY_URL environment variable.
			settings.LLMGateway.URL = strings.TrimRight(os.Getenv("LLM_GATEWAY_URL"), "/")
			log.DefaultLogger.Warn("Could not get LLM Gateway URL from config, trying LLM_GATEWAY_URL env var", "LLM_GATEWAY_URL", settings.LLMGateway.URL)
		}
		if settings.LLMGateway.URL == "" {
			// For debugging purposes only.
			settings.LLMGateway.URL = "http://llm-gateway:4033"
			log.DefaultLogger.Warn("Could not get LLM_GATEWAY_URL, using default", "default", settings.LLMGateway.URL)
		}
	default:
		// Default to Grafana-provided OpenAI if an unknown provider was specified.
		log.DefaultLogger.Warn("Unknown OpenAI provider", "provider", settings.OpenAI.Provider)
		settings.OpenAI.Provider = openAIProviderGrafana
	}

	settings.OpenAI.apiKey = appSettings.DecryptedSecureJSONData[openAIKey]

	return &settings, nil
}
