package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type OpenAI interface {
	Enabled(ctx context.Context) (bool, error)
	ChatCompletions(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	ChatCompletionsStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}

type openAI struct {
	httpClient *http.Client
	client     *openai.Client

	grafanaURL, grafanaAPIKey string
}

func NewOpenAI(grafanaURL, grafanaAPIKey string) OpenAI {
	httpClient := &http.Client{}
	return NewOpenAIWithClient(grafanaURL, grafanaAPIKey, httpClient)
}

func NewOpenAIWithClient(grafanaURL, grafanaAPIKey string, httpClient *http.Client) OpenAI {
	url := strings.TrimRight(grafanaURL, "/") + "/api/plugins/grafana-llm-app/resources/openai/v1"
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

type healthCheckResponse struct {
	Details struct {
		OpenAIEnabled bool `json:"openAI"`
		VectorEnabled bool `json:"vector"`
	} `json:"details"`
}

func (o *openAI) Enabled(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.grafanaURL+"/api/plugins/grafana-llm-app/health", nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.grafanaAPIKey)
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("make request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	var response healthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("unmarshal response: %w", err)
	}
	return response.Details.OpenAIEnabled, nil
}

func (o *openAI) ChatCompletions(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	return o.client.CreateChatCompletion(ctx, req)
}

func (o *openAI) ChatCompletionsStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	return o.client.CreateChatCompletionStream(ctx, req)
}
