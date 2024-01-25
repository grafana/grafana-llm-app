package plugin

import (
	"encoding/json"

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

type Settings struct {
	OpenAI OpenAISettings `json:"openAI"`

	Vector vector.VectorSettings `json:"vector"`

	// LLMGatewayURL is the URL of the LLM endpoint of the machine learning backend which
	// proxies the request to our llm-gateway. This is the Grafana Managed Key LLM solution.
	LLMGatewayURL string `json:"llmGatewayUrl"`

	// LLMOptInStatus indicates if customer has enabled the Grafana Managed Key LLM.
	// If not specified, this is unmarshalled to false.
	LLMOptInStatus bool `json:"llmOptInStatus"`
}

func loadSettings(appSettings backend.AppInstanceSettings) (*Settings, error) {
	settings := Settings{
		OpenAI: OpenAISettings{
			URL:      "https://api.openai.com",
			Provider: openAIProviderOpenAI,
		},
	}
	err := json.Unmarshal(appSettings.JSONData, &settings)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, err
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
		// Override the URL to point to llm-gateway TEMPORARY!!
		// settings.LLMGatewayURL = "http://llm-gateway:4033"
	default:
		// Default to Grafana-provided OpenAI if an unknown provider was specified.
		log.DefaultLogger.Warn("Unknown OpenAI provider", "provider", settings.OpenAI.Provider)
		settings.OpenAI.Provider = openAIProviderGrafana
	}

	settings.OpenAI.apiKey = appSettings.DecryptedSecureJSONData[openAIKey]

	return &settings, nil
}
