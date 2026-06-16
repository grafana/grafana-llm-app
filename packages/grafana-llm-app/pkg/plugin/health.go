package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/build/buildinfo"
	"github.com/sashabaranov/go-openai"
)

var supportedModels = []Model{ModelBase, ModelLarge}

type modelHealth struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Response any    `json:"response,omitempty"`
}

type llmProviderHealthDetails struct {
	Configured bool                  `json:"configured"`
	OK         bool                  `json:"ok"`
	Error      string                `json:"error,omitempty"`
	Response   any                   `json:"response,omitempty"`
	Models     map[Model]modelHealth `json:"models"`
}

type vectorHealthDetails struct {
	Enabled bool   `json:"enabled"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

type healthCheckDetails struct {
	LLMProvider llmProviderHealthDetails `json:"llmProvider"`
	// clone of LLMProvider health details for backwards compatibility (< 0.13.0)
	OpenAI  llmProviderHealthDetails `json:"openAI"`
	Vector  vectorHealthDetails      `json:"vector"`
	Version string                   `json:"version"`
}

func getVersion() string {
	buildInfo, err := buildinfo.GetBuildInfo()
	if err != nil {
		return "unknown"
	}
	return buildInfo.Version
}

func extractErrorResponse(err error) any {
	var reqErr *openai.RequestError
	if errors.As(err, &reqErr) {
		if reqErr.HTTPStatusCode > 0 {
			var responseBody any
			if len(reqErr.Body) > 0 {
				if jsonErr := json.Unmarshal(reqErr.Body, &responseBody); jsonErr == nil {
					return responseBody
				}
			}
			return map[string]any{
				"status_code": reqErr.HTTPStatusCode,
				"status":      reqErr.HTTPStatus,
				"error":       reqErr.Error(),
			}
		}
	}

	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		return map[string]any{
			"error": map[string]any{
				"message": apiErr.Message,
				"type":    apiErr.Type,
				"code":    apiErr.Code,
				"param":   apiErr.Param,
			},
		}
	}

	return nil
}

// getUnconfiguredError returns a specific error message based on the provider type
func (a *App) getUnconfiguredError() string {
	provider := a.settings.getEffectiveProvider()

	switch provider {
	case ProviderTypeAnthropic:
		return "Anthropic API key is not configured"
	case ProviderTypeOpenAI:
		return "OpenAI API key is not configured"
	case ProviderTypeAzure:
		hasAPIKey := a.settings.OpenAI.apiKey != ""
		hasMappings := len(a.settings.OpenAI.AzureMapping) > 0

		if !hasAPIKey && !hasMappings {
			return "Azure OpenAI API key and model mappings are not configured"
		}
		if !hasAPIKey {
			return "Azure OpenAI API key is not configured"
		}
		if !hasMappings {
			return "Azure model mappings are not configured"
		}
		return "Azure OpenAI configuration is incomplete"
	default:
		return "LLM provider not configured"
	}
}

func (a *App) testProviderModel(ctx context.Context, model Model) error {
	llmProvider, err := createProvider(a.settings)
	if err != nil {
		return err
	}

	req := ChatCompletionRequest{
		Model: model,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: "Hello"},
			},
		},
	}
	_, err = llmProvider.ChatCompletion(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

// llmProviderHealth checks the health of the LLM provider configuration and caches the
// result if successful. The caller must lock a.healthCheckMutex.
func (a *App) llmProviderHealth(ctx context.Context) (llmProviderHealthDetails, error) {
	if a.healthLLMProvider != nil {
		return *a.healthLLMProvider, nil
	}

	// If LLM provider is disabled it has been configured but cannot be queried.
	if a.settings.Disabled {
		return llmProviderHealthDetails{
			OK:         false,
			Configured: true,
			Error:      "LLM functionality is disabled",
		}, nil
	}

	d := llmProviderHealthDetails{
		OK:         true,
		Configured: a.settings.Configured(),
		Models:     map[Model]modelHealth{},
	}

	for _, model := range supportedModels {
		health := modelHealth{OK: false, Error: a.getUnconfiguredError()}
		if d.Configured {
			health.OK = true
			health.Error = ""
			err := a.testProviderModel(ctx, model)
			if err != nil {
				health.OK = false
				health.Error = err.Error()
				health.Response = extractErrorResponse(err)
			}
		}
		d.Models[model] = health
	}
	anyOK := false
	for _, v := range d.Models {
		if v.OK {
			anyOK = true
			break
		}
	}
	if !anyOK {
		d.OK = false
		d.Error = "No functioning models are available"

		var firstErrorResponse any
		for _, v := range d.Models {
			if !v.OK && v.Response != nil {
				firstErrorResponse = v.Response
				break
			}
		}
		d.Response = firstErrorResponse
	}

	// Only cache result if provider is ok to use.
	if d.OK {
		a.healthLLMProvider = &d
	}
	return d, nil
}

// testVectorService checks the health of VectorAPI and caches the result if
// successful. The caller must lock a.healthCheckMutex.
func (a *App) testVectorService(ctx context.Context) error {
	if a.vectorService == nil {
		return fmt.Errorf("vector service not configured")
	}
	err := a.vectorService.Health(ctx)
	if err != nil {
		return fmt.Errorf("vector service health check failed: %w", err)
	}
	return nil
}

func (a *App) vectorHealth(ctx context.Context) vectorHealthDetails {
	if a.healthVector != nil {
		return *a.healthVector
	}

	d := vectorHealthDetails{
		Enabled: a.settings.Vector.Enabled,
		OK:      true,
	}
	if !d.Enabled {
		d.OK = false
		return d
	}
	err := a.testVectorService(ctx)
	if err != nil {
		d.OK = false
		d.Error = err.Error()
	}

	// Only cache if the health check succeeded.
	if d.OK {
		a.healthVector = &d
	}
	return d
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// It returns whether each feature is working based on the plugin settings.
func (a *App) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	a.healthCheckMutex.Lock()
	defer a.healthCheckMutex.Unlock()

	provider, err := a.llmProviderHealth(ctx)
	if err != nil {
		provider.OK = false
		provider.Error = err.Error()
		provider.Response = extractErrorResponse(err)
	}

	vector := a.vectorHealth(ctx)
	if vector.Error == "" {
		a.healthVector = &vector
	}

	details := healthCheckDetails{
		LLMProvider: provider,
		OpenAI:      provider,
		Vector:      vector,
		Version:     getVersion(),
	}
	body, err := json.Marshal(details)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "failed to marshal details",
		}, nil
	}
	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusOk,
		JSONDetails: body,
	}, nil
}
