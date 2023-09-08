package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/llm/pkg/plugin/vector/embed"
	"github.com/grafana/llm/pkg/plugin/vector/store"
)

const openAIKey = "openAIKey"

type OpenAISettings struct {
	URL            string `json:"url"`
	OrganizationID string `json:"organizationId"`
	apiKey         string
}

type Settings struct {
	OpenAI OpenAISettings `json:"openAI"`

	openAIKey string

	EmbeddingSettings   embed.Settings `json:"embeddings"`
	VectorStoreSettings store.Settings `json:"vectorStore"`
}

func loadSettings(appSettings backend.AppInstanceSettings) Settings {
	settings := Settings{
		OpenAI: OpenAISettings{
			URL: "https://api.openai.com",
		},
	}
	_ = json.Unmarshal(appSettings.JSONData, &settings)

	settings.openAIKey = appSettings.DecryptedSecureJSONData[openAIKey]
	settings.EmbeddingSettings.OpenAI.APIKey = settings.openAIKey
	return settings
}
