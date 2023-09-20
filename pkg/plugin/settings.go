package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const openAIKey = "openAIKey"
const azureOpenAIKey = "azureOpenAIKey"

type OpenAISettings struct {
	URL            string `json:"url"`
	OrganizationID string `json:"organizationId"`
	apiKey         string
}

type AzureOpenAISettings struct {
	ResourceName string `json:"resource"`
	apiKey       string
}

type Settings struct {
	OpenAI      OpenAISettings      `json:"openAI"`
	AzureOpenAI AzureOpenAISettings `json:"azureOpenAI"`

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
	settings.AzureOpenAI.apiKey = appSettings.DecryptedSecureJSONData[azureOpenAIKey]
	return settings
}
