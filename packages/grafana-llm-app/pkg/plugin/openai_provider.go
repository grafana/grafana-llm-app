package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

type openAI struct {
	settings OpenAISettings
	models   *ModelSettings
	oc       *openai.Client
}

func NewOpenAIProvider(settings OpenAISettings, models *ModelSettings) (LLMProvider, error) {
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}
	cfg := openai.DefaultConfig(settings.apiKey)
	base, err := url.JoinPath(settings.URL, "/v1")
	if err != nil {
		return nil, fmt.Errorf("join url: %w", err)
	}
	cfg.BaseURL = base
	cfg.HTTPClient = client
	cfg.OrgID = settings.OrganizationID
	return &openAI{
		settings: settings,
		models:   models,
		oc:       openai.NewClientWithConfig(cfg),
	}, nil
}

func (p *openAI) Models(ctx context.Context) (ModelResponse, error) {
	return ModelResponse{
		Data: []ModelInfo{
			{ID: ModelBase},
			{ID: ModelLarge},
		},
	}, nil
}

type openAIChatCompletionRequest struct {
	ChatCompletionRequest
	// Override the model field to just be a string rather than our custom Model type.
	Model string `json:"model"`
}

func (p *openAI) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toOpenAI(p.models)
	resp, err := p.oc.CreateChatCompletion(ctx, r)
	if err != nil {
		log.DefaultLogger.Error("error creating openai chat completion", "err", err)
		return openai.ChatCompletionResponse{}, err
	}
	return resp, nil
}

func (p *openAI) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toOpenAI(p.models)
	return streamOpenAIRequest(ctx, r, p.oc)
}

func streamOpenAIRequest(ctx context.Context, r openai.ChatCompletionRequest, oc *openai.Client) (<-chan ChatCompletionStreamResponse, error) {
	r.Stream = true
	stream, err := oc.CreateChatCompletionStream(ctx, r)
	if err != nil {
		log.DefaultLogger.Error("error establishing stream", "err", err)
		return nil, err
	}
	c := make(chan ChatCompletionStreamResponse)

	go func() {
		defer stream.Close()
		defer close(c)
		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				log.DefaultLogger.Error("openai stream error", "err", err)
				c <- ChatCompletionStreamResponse{Error: err}
				return
			}

			c <- ChatCompletionStreamResponse{ChatCompletionStreamResponse: resp}
		}
	}()
	return c, nil
}
