package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

const (
	DefaultMaxCompletionTokens = 4000
)

// anthropicProvider implements the LLMProvider interface using Anthropic's OpenAI-compatible API.
// See: https://docs.anthropic.com/en/api/openai-sdk
type anthropicProvider struct {
	settings AnthropicSettings
	models   *ModelSettings
	client   *openai.Client
}

func NewAnthropicProvider(settings AnthropicSettings, models *ModelSettings) (LLMProvider, error) {
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}
	config := openai.DefaultConfig(settings.apiKey)
	base, err := url.JoinPath(settings.URL, "/v1")
	if err != nil {
		return nil, fmt.Errorf("join url: %w", err)
	}
	config.BaseURL = base
	config.HTTPClient = client

	return &anthropicProvider{
		settings: settings,
		models:   models,
		client:   openai.NewClientWithConfig(config),
	}, nil
}

func (p *anthropicProvider) Models(ctx context.Context) (ModelResponse, error) {
	return ModelResponse{
		Data: []ModelInfo{
			{ID: ModelBase},
			{ID: ModelLarge},
		},
	}, nil
}

func (p *anthropicProvider) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toAnthropic(p.models)
	log.DefaultLogger.Debug("model", "model", r.Model)

	// Anthropic requires a max tokens value
	if r.MaxTokens == 0 && r.MaxCompletionTokens == 0 {
		r.MaxCompletionTokens = DefaultMaxCompletionTokens
	}

	ForceUserMessage(&r)

	resp, err := p.client.CreateChatCompletion(ctx, r)
	if err != nil {
		log.DefaultLogger.Error("error creating anthropic chat completion", "err", err)
		return openai.ChatCompletionResponse{}, err
	}

	return resp, nil
}

func (p *anthropicProvider) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toAnthropic(p.models)
	log.DefaultLogger.Debug("model", "model", r.Model)

	// Anthropic requires a max tokens value
	if r.MaxTokens == 0 && r.MaxCompletionTokens == 0 {
		r.MaxCompletionTokens = DefaultMaxCompletionTokens
	}

	ForceUserMessage(&r)

	return streamOpenAIRequest(ctx, r, p.client)
}
