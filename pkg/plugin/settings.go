package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const openAIKey = "openAIKey"

type OpenAISettings struct {
	URL            string `json:"url"`
	OrganizationID string `json:"organizationId"`
	apiKey         string
}

type Settings struct {
	OpenAI OpenAISettings `json:"openAI"`

	Vector vector.VectorSettings `json:"vector"`
}

func loadSettings(appSettings backend.AppInstanceSettings) Settings {
	settings := Settings{
		OpenAI: OpenAISettings{
			URL: "https://api.openai.com",
		},
	}
	_ = json.Unmarshal(appSettings.JSONData, &settings)

	settings.OpenAI.apiKey = appSettings.DecryptedSecureJSONData[openAIKey]
	return settings
}
