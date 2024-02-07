package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/embed"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const (
	providerOpenAI providerName = "openai"
	providerAzure  providerName = "azure"
	providerPulze  providerName = "pulze"

	providerKey = "providerKey"
)

type providerName string

type ProviderSettings struct {
	URL            string       `json:"url"`
	OrganizationID string       `json:"organizationId"`
	Name           providerName `json:"name"`
	// Deprecated: Use Name instead.
	Provider     providerName `json:"provider"`
	AzureMapping [][]string   `json:"azureModelMapping"`
	PulzeModel   string       `json:"pulzeModel"`
	apiKey       string
}

type Settings struct {
	// Deprecated: Use Provider instead.
	OpenAI   ProviderSettings      `json:"openAI"`
	Provider ProviderSettings      `json:"provider"`
	Vector   vector.VectorSettings `json:"vector"`
}

func loadSettings(appSettings backend.AppInstanceSettings) Settings {
	settings := Settings{
		Provider: ProviderSettings{
			URL:  "https://api.openai.com",
			Name: providerOpenAI,
		},
	}

	_ = json.Unmarshal(appSettings.JSONData, &settings)
	// populate the new settings structure from 'openAI' if not present
	if settings.Provider.Name == "" {
		settings.Provider = settings.OpenAI
		settings.Provider.Name = settings.Provider.Provider
	}

	if settings.Vector.Embed.Type == embed.EmbedderOpenAI {
		settings.Vector.Embed.OpenAI.URL = settings.Provider.URL
		settings.Vector.Embed.OpenAI.AuthType = "openai-key-auth"
	}

	switch settings.Provider.Name {
	case providerOpenAI:
		// We need to handle the case where the user has customized the URL,
		// then reverted that customization so that the JSON data includes
		// an empty string.
		if settings.Provider.URL == "" {
			settings.Provider.URL = "https://api.openai.com"
		}
	case providerAzure:
	case providerPulze:
		if settings.Provider.URL == "" {
			settings.Provider.URL = "https://api.pulze.ai"
		}
	default:
		// Default to OpenAI if an unknown provider was specified.
		settings.Provider.Name = providerOpenAI
	}

	settings.Provider.apiKey = appSettings.DecryptedSecureJSONData[providerKey]
	if settings.Provider.apiKey == "" {
		log.DefaultLogger.Warn("[DecryptedSecureJSONData] no 'providerKey' given, using 'openAIKey'")
		settings.Provider.apiKey = appSettings.DecryptedSecureJSONData["openAIKey"]
	}

	return settings
}
