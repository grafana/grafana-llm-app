package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/build"
	"github.com/sashabaranov/go-openai"
)

var supportedModels = []Model{ModelBase, ModelLarge}

type modelHealth struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type llmProviderHealthDetails struct {
	Configured bool                  `json:"configured"`
	OK         bool                  `json:"ok"`
	Error      string                `json:"error,omitempty"`
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
	buildInfo, err := build.GetBuildInfo()
	if err != nil {
		return "unknown"
	}
	return buildInfo.Version
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
			MaxTokens: 1,
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
		health := modelHealth{OK: false, Error: "LLM provider not configured"}
		if d.Configured {
			health.OK = true
			health.Error = ""
			err := a.testProviderModel(ctx, model)
			if err != nil {
				health.OK = false
				health.Error = err.Error()
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
