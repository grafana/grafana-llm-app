package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// DefaultGrafanaLLMAppConfig creates new OpenAI client config for use
// when accessing OpenAI via the Grafana LLM App.
func DefaultGrafanaLLMAppConfig(grafanaURL, grafanaAPIKey string) openai.ClientConfig {
	url := strings.TrimRight(grafanaURL, "/") + "/api/plugins/grafana-llm-app/resources/openai/v1"
	cfg := openai.DefaultConfig(grafanaAPIKey)
	cfg.BaseURL = url
	return cfg
}

type healthCheckResponse struct {
	OpenAIEnabled bool `json:"openAI"`
	VectorEnabled bool `json:"vector"`
}

// ErrPluginNotInstalled is returned when the Grafana LLM App plugin is not installed.
var ErrPluginNotInstalled = errors.New("grafana-llm-app plugin not installed")

func checkHealth(ctx context.Context, grafanaURL, grafanaAPIKey string) (*healthCheckResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", grafanaURL+"/api/plugins/grafana-llm-app/health", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+grafanaAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrPluginNotInstalled
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check health: %d", resp.StatusCode)
	}
	var details healthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &details, nil
}

// OpenAIEnabled returns true if the Grafana LLM App is configured to use OpenAI.
func OpenAIEnabled(ctx context.Context, grafanaURL, grafanaAPIKey string) (bool, error) {
	details, err := checkHealth(ctx, grafanaURL, grafanaAPIKey)
	if err != nil {
		return false, err
	}
	return details.OpenAIEnabled, nil
}
