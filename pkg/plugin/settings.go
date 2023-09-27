package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const openAIKey = "openAIKey"

type openAIProvider string

const (
	openAIProviderOpenAI openAIProvider = "openai"
	openAIProviderAzure  openAIProvider = "azure"
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
}

func loadSettings(appSettings backend.AppInstanceSettings) Settings {
	settings := Settings{
		OpenAI: OpenAISettings{
			URL:      "https://api.openai.com",
			Provider: openAIProviderOpenAI,
		},
	}
	_ = json.Unmarshal(appSettings.JSONData, &settings)

	switch settings.OpenAI.Provider {
	case openAIProviderOpenAI:
	case openAIProviderAzure:
	default:
		// Default to OpenAI if an unknown provider was specified.
		log.DefaultLogger.Warn("Unknown OpenAI provider", "provider", settings.OpenAI.Provider)
		settings.OpenAI.Provider = openAIProviderOpenAI
	}

	settings.OpenAI.apiKey = appSettings.DecryptedSecureJSONData[openAIKey]
	return settings
}
