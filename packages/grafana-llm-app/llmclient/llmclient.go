// package llmclient provides a client for the Grafana LLM app.
// It is used to communicate with LLM providers via the Grafana LLM app
// using the configuration stored in the app to handle authentication.
package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

const (
	appPrefix          = "/api/plugins/grafana-llm-app"
	appResourcesPrefix = appPrefix + "/resources"
	llmAPIPrefix       = appResourcesPrefix + "/llm/v1"
)

// Model is an abstraction over the different models available in different providers
type Model string

const (
	// ModelBase is the base model, for efficient and high-throughput tasks
	ModelBase = "base"
	// ModelLarge is the large model, for more advanced tasks with longer context windows
	ModelLarge = "large"
)

// ChatCompletionRequest is a request for chat completions using an abstract model.
type ChatCompletionRequest struct {
	openai.ChatCompletionRequest
	Model Model `json:"model"`
}

// LLMProvider is an interface for talking to LLM providers via the Grafana LLM app.
// Requests made using this interface will be routed to the configured LLM provider backend
// with authentication handled by the LLM app.
type LLMProvider interface {
	// Enabled returns true if the Grafana LLM app has been configured for use
	// with an LLM provider.
	Enabled(ctx context.Context) (bool, error)
	// ChatCompletions makes a request to the LLM provider Chat Completion API.
	ChatCompletions(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	// ChatCompletionsStream makes a streaming request to the LLM provider Chat Completion API.
	ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}

type llmProvider struct {
	httpClient *http.Client
	client     *openai.Client

	grafanaURL, grafanaAPIKey string
}

// NewLLMProvider creates a new LLM provider client talking to the Grafana LLM app installed
// on the given Grafana instance.
func NewLLMProvider(grafanaURL, grafanaAPIKey string) LLMProvider {
	httpClient := &http.Client{}
	return NewLLMProviderWithClient(grafanaURL, grafanaAPIKey, httpClient)
}

// NewLLMProviderWithClient creates a new LLM provider client talking to the Grafana LLM app installed
// on the given Grafana instance, using the given HTTP client.
func NewLLMProviderWithClient(grafanaURL, grafanaAPIKey string, httpClient *http.Client) LLMProvider {
	grafanaURL = strings.TrimRight(grafanaURL, "/")
	url := grafanaURL + llmAPIPrefix
	cfg := openai.DefaultConfig(grafanaAPIKey)
	cfg.BaseURL = url
	cfg.HTTPClient = httpClient
	client := openai.NewClientWithConfig(cfg)
	return &llmProvider{
		httpClient:    httpClient,
		client:        client,
		grafanaURL:    grafanaURL,
		grafanaAPIKey: grafanaAPIKey,
	}
}

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
	Vector      vectorHealthDetails      `json:"vector"`
	Version     string                   `json:"version"`
}

type healthCheckResponse struct {
	Details healthCheckDetails `json:"details"`
}

type oldHealthCheckResponse struct {
	Details struct {
		LLMProviderEnabled bool `json:"llmProvider"`
		VectorEnabled      bool `json:"vector"`
	} `json:"details"`
}

func (o *llmProvider) Enabled(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.grafanaURL+appPrefix+"/health", nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.grafanaAPIKey)
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read response body: %w", err)
	}
	var response healthCheckResponse
	if err := json.Unmarshal(body, &response); err != nil {
		// Try the old response format
		var oldResponse oldHealthCheckResponse
		if err := json.Unmarshal(body, &oldResponse); err != nil {
			return false, fmt.Errorf("unmarshal response: %w", err)
		}
		return oldResponse.Details.LLMProviderEnabled, nil
	}
	if response.Details.LLMProvider.Error != "" {
		err = fmt.Errorf("LLM provider error: %s", response.Details.LLMProvider.Error)
	}
	return response.Details.LLMProvider.OK, err
}

func (o *llmProvider) ChatCompletions(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = string(req.Model)
	return o.client.CreateChatCompletion(ctx, r)
}

func (o *llmProvider) ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	r := req.ChatCompletionRequest
	r.Model = string(req.Model)
	return o.client.CreateChatCompletionStream(ctx, r)
}
