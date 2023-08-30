package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/llm/pkg/plugin/vector/embed"
	"github.com/grafana/llm/pkg/plugin/vector/store"
)

const openAIKey = "openAIKey"

type Settings struct {
	OpenAIURL            string `json:"openAIUrl"`
	OpenAIOrganizationID string `json:"openAIOrganizationId"`

	openAIKey string

	EmbeddingSettings   embed.Settings `json:"embeddings"`
	VectorStoreSettings store.Settings `json:"vectorStore"`
}

func loadSettings(appSettings backend.AppInstanceSettings) Settings {
	settings := Settings{
		OpenAIURL: "https://api.openai.com",
	}
	_ = json.Unmarshal(appSettings.JSONData, &settings)

	settings.openAIKey = appSettings.DecryptedSecureJSONData[openAIKey]
	settings.EmbeddingSettings.OpenAI.APIKey = settings.openAIKey
	return settings
}
