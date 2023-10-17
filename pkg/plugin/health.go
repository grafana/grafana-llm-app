package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/build"
)

var openAIModels = []string{"gpt-3.5-turbo", "gpt-4"}

type healthCheckClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type openAIModelHealth struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type openAIHealthDetails struct {
	Configured bool                         `json:"configured"`
	OK         bool                         `json:"ok"`
	Error      string                       `json:"error,omitempty"`
	Models     map[string]openAIModelHealth `json:"models"`
}

type vectorHealthDetails struct {
	Enabled bool   `json:"enabled"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

type healthCheckDetails struct {
	OpenAI  openAIHealthDetails `json:"openAI"`
	Vector  vectorHealthDetails `json:"vector"`
	Version string              `json:"version"`
}

func getVersion() string {
	buildInfo, err := build.GetBuildInfo()
	if err != nil {
		return "unknown"
	}
	return buildInfo.Version
}

func (a *App) testOpenAIModel(ctx context.Context, url *url.URL, model string) error {
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "Hello",
			},
		},
	}
	req, err := a.newOpenAIChatCompletionsRequest(ctx, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := a.healthCheckClient.Do(req)
	if err != nil {
		return fmt.Errorf("make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

func (a *App) openAIHealth(ctx context.Context) (openAIHealthDetails, error) {
	d := openAIHealthDetails{
		OK:         true,
		Configured: a.settings.OpenAI.apiKey != "",
		Models:     map[string]openAIModelHealth{},
	}
	u, err := url.Parse(a.settings.OpenAI.URL)
	if err != nil {
		return d, fmt.Errorf("Unable to parse OpenAI URL: %w", err)
	}

	for _, model := range openAIModels {
		health := openAIModelHealth{OK: false, Error: "OpenAI not configured"}
		if d.Configured {
			health.OK = true
			health.Error = ""
			err := a.testOpenAIModel(ctx, u, model)
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
		d.Error = "No models are working"
	}
	return d, nil
}

func (a *App) testVectorService(ctx context.Context) (bool, error) {
	if a.vectorService == nil {
		return false, fmt.Errorf("vector service not configured")
	}
	result, err := a.vectorService.Health(ctx)
	return result, err
}

func (a *App) vectorHealth(ctx context.Context) vectorHealthDetails {
	d := vectorHealthDetails{
		Enabled: a.settings.Vector.Enabled,
		OK:      true,
	}
	if !d.Enabled {
		d.OK = false
		return d
	}
	result, err := a.testVectorService(ctx)
	if err != nil {
		d.OK = false
		d.Error = err.Error()
	}
	if !result {
		d.OK = false
		d.Error = "Vector service health check failed"
	}
	return d
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// It returns whether each feature is working based on the plugin settings.
func (a *App) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	a.healthCheckMutex.Lock()
	defer a.healthCheckMutex.Unlock()
	if a.healthCheckResult != nil {
		return a.healthCheckResult, nil
	}
	openAI, err := a.openAIHealth(ctx)
	if err != nil {
		openAI.OK = false
		openAI.Error = err.Error()
	}
	vector := a.vectorHealth(ctx)
	details := healthCheckDetails{
		OpenAI:  openAI,
		Vector:  vector,
		Version: getVersion(),
	}
	body, err := json.Marshal(details)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "failed to marshal details",
		}, nil
	}
	a.healthCheckResult = &backend.CheckHealthResult{
		Status:      backend.HealthStatusOk,
		JSONDetails: body,
	}
	return a.healthCheckResult, nil
}
