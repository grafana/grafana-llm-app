package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func (a *App) newAuthenticatedOpenAIRequest(ctx context.Context, method string, url url.URL, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), body)
	if err != nil {
		return nil, err
	}
	switch a.settings.Provider.Provider {
	case openAIProviderOpenAI:
		req.Header.Set("Authorization", "Bearer "+a.settings.Provider.apiKey)
		req.Header.Set("OpenAI-Organization", a.settings.Provider.OrganizationID)
	case openAIProviderAzure:
		req.Header.Set("api-key", a.settings.Provider.apiKey)
	case openAIProviderPulze:
		log.DefaultLogger.Debug("In newAuthenticatedOpenAIRequest case")
		req.Header.Set("Authorization", "Bearer "+a.settings.Provider.apiKey)
		log.DefaultLogger.Debug(a.settings.Provider.apiKey)
		pulzeLabels := fmt.Sprintf("{\"grafana_org_id\": \"%s\"}", a.settings.Provider.OrganizationID)
		req.Header.Set("Pulze-Labels", pulzeLabels)
	}
	return req, nil
}

func (a *App) newOpenAIChatCompletionsRequest(ctx context.Context, openAIURL *url.URL, body map[string]interface{}) (*http.Request, error) {
	log.DefaultLogger.Debug("Receiving OpenAIChatCompletionsRequest")
	url := openAIURL
	switch a.settings.Provider.Provider {
	case openAIProviderOpenAI:
		url.Path = "/v1/chat/completions"

	case openAIProviderAzure:
		deployment := ""
		for _, v := range a.settings.Provider.AzureMapping {
			if val, ok := body["model"].(string); ok && val == v[0] {
				deployment = v[1]
				break
			}
		}
		if deployment == "" {
			return nil, fmt.Errorf("no deployment found for model: %s", body["model"])
		}
		delete(body, "model")
		url.Path = fmt.Sprintf("/openai/deployments/%s/chat/completions", deployment)
		url.RawQuery = "api-version=2023-03-15-preview"
	case openAIProviderPulze:
		log.DefaultLogger.Debug("Receiving OpenAIChatCompletionsRequest: in pulze case")
		url.Path = "/v1/chat/completions"

	default:
		return nil, fmt.Errorf("Unknown provider: %s", a.settings.Provider.Provider)
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}
	req, err := a.newAuthenticatedOpenAIRequest(ctx, http.MethodPost, *url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
