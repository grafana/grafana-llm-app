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

// Define models for each provider.
var providerModels = map[string][]string{
	"openAI": []string{"gpt-3.5-turbo", "gpt-4"},
	"pulze":  []string{"openai/gpt-4", "pulze"},
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
	Provider map[string]providerHealthDetails `json:"provider"`
	Vector   vectorHealthDetails              `json:"vector"`
	Version  string                           `json:"version"`
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
	log.DefaultLogger.Debug("!!!! In testModel !!!!")

	if url == nil {
		log.DefaultLogger.Error("URL is nil in testModel")
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
	log.DefaultLogger.Debug("!!!! Request done !!!!")
	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.DefaultLogger.Debug(fmt.Sprintf("error in status code: %#s", resp))
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		log.DefaultLogger.Debug(fmt.Sprintf("error in status code %d: %s", resp.StatusCode, respBody))
		return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, respBody)
	}
	log.DefaultLogger.Debug(fmt.Sprintf("Request received: %#s", resp))
	return nil
}

// checkProviderHealth performs a health check for a specific provider.
func (a *App) checkProviderHealth(ctx context.Context, provider string, models []string) (providerHealthDetails, error) {
	if a.healthProvider != nil {
		return *a.healthProvider, nil
	}
	log.DefaultLogger.Debug(fmt.Sprintf("in checkProviderHealth with %#s", models))

	d := providerHealthDetails{
		Configured: a.settings.Provider.apiKey != "",
		OK:         true,
		Models:     make(map[string]modelHealth),
	}
	u, err := url.Parse(a.settings.Provider.URL)
	if err != nil {
		return d, fmt.Errorf("Unable to parse Provider URL: %w", err)
	}

	for _, model := range models {
		health := modelHealth{OK: true}
		err := a.testModel(ctx, u, model) // Pass the appropriate URL for each provider
		if err != nil {
			health.OK = false
			health.Error = err.Error()
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

	// Only cache result if openAI is ok to use.
	if d.OK {
		a.healthProvider = &d
	}
	return d, nil
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

// vectorHealth performs a health check for the Vector service.
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

// CheckHealth handles health checks for all providers and the Vector service.
func (a *App) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	a.healthCheckMutex.Lock()
	defer a.healthCheckMutex.Unlock()

	details := healthCheckDetails{
		Provider: make(map[string]providerHealthDetails),
		Vector:   a.vectorHealth(ctx),
		Version:  getVersion(),
	}

	// Iterate through each provider and perform health checks.
	for provider, models := range providerModels {
		log.DefaultLogger.Debug(fmt.Sprintf("@@@ %#s %#s", provider, models))
		log.DefaultLogger.Debug(fmt.Sprintf("@@@ %#s", a.settings.Provider.Provider))
		if provider == string(a.settings.Provider.Provider) {
			providerHealth, err := a.checkProviderHealth(ctx, provider, models)
			if err != nil {
				log.DefaultLogger.Error(fmt.Sprintf("Health check failed for provider %s: %s", provider, err))
				// Handle the error according to your application logic.
				// For instance, you can continue with the next provider or abort the entire process.
				continue
			}
			details.Provider[provider] = providerHealth
		}
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
