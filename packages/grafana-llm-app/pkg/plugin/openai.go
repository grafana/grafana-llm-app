package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

func (a *App) newAuthenticatedOpenAIRequest(ctx context.Context, method string, url url.URL, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), body)
	if err != nil {
		return nil, err
	}
	switch a.settings.OpenAI.Provider {
	case openAIProviderOpenAI:
		req.Header.Set("Authorization", "Bearer "+a.settings.OpenAI.apiKey)
		req.Header.Set("OpenAI-Organization", a.settings.OpenAI.OrganizationID)
	case openAIProviderAzure:
		req.Header.Set("api-key", a.settings.OpenAI.apiKey)
	case openAIProviderGrafana:
		req.SetBasicAuth(a.settings.Tenant, a.settings.GrafanaComAPIKey)
		req.Header.Add("X-Scope-OrgID", a.settings.Tenant)
	}
	return req, nil
}

func (a *App) newOpenAIChatCompletionsRequest(ctx context.Context, body map[string]interface{}) (*http.Request, error) {
	var url *url.URL
	var err error

	switch a.settings.OpenAI.Provider {
	case openAIProviderOpenAI:
		url, err = url.Parse(a.settings.OpenAI.URL)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse OpenAI URL: %w", err)
		}
		url.Path = "/v1/chat/completions"

	case openAIProviderAzure:
		deployment := ""
		for _, v := range a.settings.OpenAI.AzureMapping {
			if val, ok := body["model"].(string); ok && val == v[0] {
				deployment = v[1]
				break
			}
		}
		if deployment == "" {
			return nil, fmt.Errorf("no deployment found for model: %s", body["model"])
		}
		delete(body, "model")

		url, err = url.Parse(a.settings.OpenAI.URL)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse OpenAI URL: %w", err)
		}
		url.Path = fmt.Sprintf("/openai/deployments/%s/chat/completions", deployment)
		url.RawQuery = "api-version=2023-03-15-preview"

	case openAIProviderGrafana:
		url, err = url.Parse(a.settings.LLMGateway.URL)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse LLM Gateway URL: %w", err)
		}
		url.Path = path.Join(url.Path, "/openai/v1/chat/completions")

	default:
		return nil, fmt.Errorf("Unknown OpenAI provider: %s", a.settings.OpenAI.Provider)
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
