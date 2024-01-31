package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/build"
)

// Define models for each provider to be included in the health check.
var providerModels = map[string][]string{
	"openai": {"gpt-3.5-turbo", "gpt-4"},
	"pulze":  {"pulze", "openai/gpt-4"},
}

type healthCheckClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type modelHealth struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type providerHealthDetails struct {
	Configured bool                   `json:"configured"`
	OK         bool                   `json:"ok"`
	Error      string                 `json:"error,omitempty"`
	Models     map[string]modelHealth `json:"models"`
}

type vectorHealthDetails struct {
	Enabled bool   `json:"enabled"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

type healthCheckDetails struct {
	Provider providerHealthDetails `json:"provider"`
	Vector   vectorHealthDetails   `json:"vector"`
	Version  string                `json:"version"`
}

func getVersion() string {
	buildInfo, err := build.GetBuildInfo()
	if err != nil {
		return "unknown"
	}
	return buildInfo.Version
}

// testModel simulates a health check for a specific model of a provider.
func (a *App) testModel(ctx context.Context, url *url.URL, model string) error {
	if url == nil {
		return fmt.Errorf("URL is nil")
	}

	// Simulated health check logic goes here
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

// providerHealth performs a health check for the selected provider and caches the
// result if successful. The caller must lock a.healthCheckMutex.
func (a *App) providerHealth(ctx context.Context, provider string, models []string) providerHealthDetails {
	if a.healthProvider != nil {
		log.DefaultLogger.Debug("returning cached healthProvider:", a.healthProvider.Models)
		return *a.healthProvider
	}
	log.DefaultLogger.Debug(fmt.Sprintf("in checkProviderHealth with %s", models))

	d := providerHealthDetails{
		Configured: a.settings.Provider.apiKey != "",
		OK:         true,
		Models:     make(map[string]modelHealth),
	}
	u, err := url.Parse(a.settings.Provider.URL)
	if err != nil {
		d.OK = false
		d.Error = fmt.Sprintf("Unable to parse provider URL: %s", err)
		return d
	}

	for _, model := range models {
		health := modelHealth{OK: false, Error: "model not configured."}
		if d.Configured {
			health.OK = true
			health.Error = ""
			err := a.testModel(ctx, u, model)
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

	// Only cache result if provider is ok to use.
	if d.OK {
		a.healthProvider = &d
	}
	return d
}

// testVectorService checks the health of VectorAPI and caches the result if successful.
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

// vectorHealth performs a health check for the Vector service and caches the
// result if successful. The caller must lock a.healthCheckMutex.
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

// CheckHealth handles health checks for the selected provider and the Vector service.
func (a *App) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	a.healthCheckMutex.Lock()
	defer a.healthCheckMutex.Unlock()

	log.DefaultLogger.Debug("CheckHealth", a.settings.Provider.Name, a.settings.Provider)

	ps := string(a.settings.Provider.Name)
	log.DefaultLogger.Debug("")
	provider := a.providerHealth(ctx, ps, providerModels[ps])
	if provider.Error == "" {
		a.healthProvider = &provider
	}

	vector := a.vectorHealth(ctx)
	if vector.Error == "" {
		a.healthVector = &vector
	}
	details := healthCheckDetails{
		Provider: provider,
		Vector:   vector,
		Version:  getVersion(),
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
