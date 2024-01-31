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
	Name           providerName `json:"name"` // TODO: make use of this field and drop "Provider"
	// Deprecated: Use Name instead.
	Provider     providerName `json:"provider"`
	AzureMapping [][]string   `json:"azureModelMapping"`
	apiKey       string
}

type Settings struct {
	Provider ProviderSettings      `json:"provider"`
	Vector   vector.VectorSettings `json:"vector"`
}

// UnmarshalJSON supports parsing of old settings structure.
func (s *Settings) UnmarshalJSON(data []byte) error {
	// partially parse data to check for keys
	var ps map[string]json.RawMessage
	err := json.Unmarshal(data, &ps)
	if err != nil {
		return err
	}

	// parse vector field
	err = json.Unmarshal(ps["vector"], &s.Vector)
	if err != nil {
		return err
	}

	// use 'openAI' (old settings structure) as a fallback
	keyToParse := "provider"
	if _, exists := ps[keyToParse]; !exists {
		log.DefaultLogger.Warn("No 'provider' settings found, using deprecated 'openAI' field! Please migrate to 'settings.provider'")
		keyToParse = "openAI"
	}
	return json.Unmarshal(ps[keyToParse], &s.Provider)
}

func loadSettings(appSettings backend.AppInstanceSettings) Settings {
	settings := Settings{
		Provider: ProviderSettings{
			URL:  "https://api.openai.com",
			Name: providerOpenAI,
		},
	}
	_ = json.Unmarshal(appSettings.JSONData, &settings)

	if settings.Vector.Embed.Type == embed.EmbedderOpenAI {
		settings.Vector.Embed.OpenAI.URL = settings.Provider.URL
		settings.Vector.Embed.OpenAI.AuthType = "openai-key-auth"
	}

	switch settings.Provider.Provider {
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
		log.DefaultLogger.Warn("Unknown OpenAI provider", "provider", settings.Provider.Provider)
		settings.Provider.Provider = providerOpenAI
	}

	settings.Provider.apiKey = appSettings.DecryptedSecureJSONData[providerKey]
	if settings.Provider.apiKey == "" {
		log.DefaultLogger.Warn("[DecryptedSecureJSONData] no 'providerKey' given, using 'openAIKey'")
		settings.Provider.apiKey = appSettings.DecryptedSecureJSONData["openAIKey"]
	}

	return settings
}
