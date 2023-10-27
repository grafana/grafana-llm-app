package plugin

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestEmbeddingSettingLogic(t *testing.T) {

	// Set up and run test cases
	for _, tc := range []struct {
		name                   string
		settings               backend.AppInstanceSettings
		embeddingURL           string
		embeddingAuthType      string
		embeddingBasicAuthUser string
	}{
		{
			name: "openai-embedder-option",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"openAI": {
						"url": "https://api.openai.com",
						"provider": "openai"
					},
					"vector": {
						"embed": {
							"type": "openai"
						}
					}
				}`),
				DecryptedSecureJSONData: map[string]string{openAIKey: "abcd1234"},
			},
			embeddingURL:           "https://api.openai.com",
			embeddingAuthType:      "openai-key-auth",
			embeddingBasicAuthUser: "",
		},
		{
			name: "grafana-vector-api-no-auth",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"vector": {
						"embed": {
							"type": "grafana/vectorapi",
							"openai": {
								"url": "https://api.example.com",
								"authType": "no-auth"
							}
						}
					}
				}`),
			},
			embeddingURL:           "https://api.example.com",
			embeddingAuthType:      "no-auth",
			embeddingBasicAuthUser: "",
		},
		{
			name: "grafana-vector-api-basic-auth",
			settings: backend.AppInstanceSettings{
				JSONData: []byte(`{
					"vector": {
						"embed": {
							"type": "grafana/vectorapi",
							"openai": {
								"url": "https://api.example.com",
								"authType": "basic-auth",
								"basicAuthUser": "test"
							}
						}
					}
				}`),
			},
			embeddingURL:           "https://api.example.com",
			embeddingAuthType:      "basic-auth",
			embeddingBasicAuthUser: "test",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := loadSettings(tc.settings)

			// Assert that the settings are loaded correctly
			if settings.Vector.Embed.OpenAI.URL != tc.embeddingURL {
				t.Errorf("expected embedding URL to be %s, got %s", tc.embeddingURL, settings.Vector.Embed.OpenAI.URL)
			}

			if settings.Vector.Embed.OpenAI.AuthType != tc.embeddingAuthType {
				t.Errorf("expected embedding auth type to be %s, got %s", tc.embeddingAuthType, settings.Vector.Embed.OpenAI.AuthType)
			}

			if settings.Vector.Embed.OpenAI.BasicAuthUser != tc.embeddingBasicAuthUser {
				t.Errorf("expected embedding basic auth user to be %s, got %s", tc.embeddingBasicAuthUser, settings.Vector.Embed.OpenAI.BasicAuthUser)
			}

		})
	}
}
