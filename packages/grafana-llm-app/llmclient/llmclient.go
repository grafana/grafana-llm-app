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
)

// Model is an abstraction over the different models available in different providers
type Model string

const (
	// ModelSmall is the small model, which is the fastest and cheapest to use. OpenAI default: gpt-3.5-turbo
	ModelSmall = "small"
	// ModelMedium is the medium model, which is a good balance between speed and cost. OpenAI default: gpt-4-turbo
	ModelMedium = "medium"
	// ModelLarge is the large model, which is the most powerful and accurate. OpenAI default: gpt-4
	ModelLarge = "large"
)

type ChatCompletionRequest struct {
	openai.ChatCompletionRequest
	Model Model `json:"model"`
}

// OpenAI is an interface for talking to OpenAI via the Grafana LLM app.
// Requests made using this interface will be routed to the OpenAI backend
// configured in the Grafana LLM app's settings, with authentication handled
// by the LLM app.
type OpenAI interface {
	// Enabled returns true if the Grafana LLM app has been configured for use
	// with OpenAI.
	Enabled(ctx context.Context) (bool, error)
	// ChatCompletions makes a request to the OpenAI Chat Completion API.
	ChatCompletions(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	// ChatCompletionsStream makes a streaming request to the OpenAI Chat Completion API.
	ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}

type openAI struct {
	httpClient *http.Client
	client     *openai.Client

	grafanaURL, grafanaAPIKey string
}

// NewOpenAI creates a new OpenAI client talking to the Grafana LLM app installed
// on the given Grafana instance.
func NewOpenAI(grafanaURL, grafanaAPIKey string) OpenAI {
	httpClient := &http.Client{}
	return NewOpenAIWithClient(grafanaURL, grafanaAPIKey, httpClient)
}

// NewOpenAIWithClient creates a new OpenAI client talking to the Grafana LLM app installed
// on the given Grafana instance, using the given HTTP client.
func NewOpenAIWithClient(grafanaURL, grafanaAPIKey string, httpClient *http.Client) OpenAI {
	grafanaURL = strings.TrimRight(grafanaURL, "/")
	url := grafanaURL + appResourcesPrefix + "/openai/v1"
	cfg := openai.DefaultConfig(grafanaAPIKey)
	cfg.BaseURL = url
	cfg.HTTPClient = httpClient
	client := openai.NewClientWithConfig(cfg)
	return &openAI{
		httpClient:    httpClient,
		client:        client,
		grafanaURL:    grafanaURL,
		grafanaAPIKey: grafanaAPIKey,
	}
}

type openAIModelHealth struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type openAIHealthDetails struct {
	Configured bool                        `json:"configured"`
	OK         bool                        `json:"ok"`
	Error      string                      `json:"error,omitempty"`
	Models     map[Model]openAIModelHealth `json:"models"`
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

type healthCheckResponse struct {
	Details healthCheckDetails `json:"details"`
}

type oldHealthCheckResponse struct {
	Details struct {
		OpenAIEnabled bool `json:"openAI"`
		VectorEnabled bool `json:"vector"`
	} `json:"details"`
}

func (o *openAI) Enabled(ctx context.Context) (bool, error) {
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
		return oldResponse.Details.OpenAIEnabled, nil
	}
	if response.Details.OpenAI.Error != "" {
		err = fmt.Errorf("OpenAI error: %s", response.Details.OpenAI.Error)
	}
	return response.Details.OpenAI.OK, err
}

func (o *openAI) ChatCompletions(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = string(req.Model)
	return o.client.CreateChatCompletion(ctx, r)
}

func (o *openAI) ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	r := req.ChatCompletionRequest
	r.Model = string(req.Model)
	return o.client.CreateChatCompletionStream(ctx, r)
}
