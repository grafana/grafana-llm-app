package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/embed"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const providerKey = "providerKey"

type openAIProvider string

const (
	openAIProviderOpenAI openAIProvider = "openai"
	openAIProviderAzure  openAIProvider = "azure"
	openAIProviderPulze  openAIProvider = "pulze"
)

type ProviderSettings struct {
	URL            string         `json:"url"`
	OrganizationID string         `json:"organizationId"`
	Provider       openAIProvider `json:"provider"`
	AzureMapping   [][]string     `json:"azureModelMapping"`
	apiKey         string
	defaultModel   string
}

type Settings struct {
	Provider ProviderSettings `json:"provider"`

	Vector vector.VectorSettings `json:"vector"`
}

func loadSettings(appSettings backend.AppInstanceSettings) Settings {
	log.DefaultLogger.Debug("In loading settings")

	settings := Settings{
		Provider: ProviderSettings{
			URL:      "https://api.openai.com",
			Provider: openAIProviderOpenAI,
		},
	}
	_ = json.Unmarshal(appSettings.JSONData, &settings)

	// We need to handle the case where the user has customized the URL,
	// then reverted that customization so that the JSON data includes
	// an empty string.
	if settings.Provider.URL == "" {
		settings.Provider.URL = "https://api.openai.com"
	}
	if settings.Vector.Embed.Type == embed.EmbedderOpenAI {
		settings.Vector.Embed.OpenAI.URL = settings.Provider.URL
		settings.Vector.Embed.OpenAI.AuthType = "openai-key-auth"
	}

	switch settings.Provider.Provider {
	case openAIProviderOpenAI:
	case openAIProviderAzure:
	case openAIProviderPulze:
		log.DefaultLogger.Debug("In loading settings pulze case")
		log.DefaultLogger.Debug(fmt.Sprintf("Settings %s", settings.Provider))
		if settings.Provider.URL == "" {
			settings.Provider.URL = "https://api.pulze.ai"
		}
		if settings.Provider.defaultModel == "" {
			settings.Provider.defaultModel = "pulze-v0"
		}
	default:
		// Default to OpenAI if an unknown provider was specified.
		log.DefaultLogger.Warn("Unknown OpenAI provider", "provider", settings.Provider.Provider)
		settings.Provider.Provider = openAIProviderOpenAI
	}

	settings.Provider.apiKey = appSettings.DecryptedSecureJSONData[providerKey]

	return settings
}
