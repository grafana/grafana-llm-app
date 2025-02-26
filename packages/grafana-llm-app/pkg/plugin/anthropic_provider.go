package plugin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

type anthropicProvider struct {
	settings AnthropicSettings
	models   *ModelSettings
	client   *anthropic.Client
}

func NewAnthropicProvider(settings AnthropicSettings, models *ModelSettings) (LLMProvider, error) {
	httpClientOpts := option.WithHTTPClient(&http.Client{
		Timeout: 2 * time.Minute,
	})
	client := anthropic.NewClient(
		option.WithAPIKey(settings.apiKey),
		httpClientOpts,
	)

	if settings.URL != "" {
		client = anthropic.NewClient(
			option.WithAPIKey(settings.apiKey),
			option.WithBaseURL(settings.URL),
			httpClientOpts,
		)
	}

	defaultModels := &ModelSettings{
		Default: ModelBase,
		Mapping: map[Model]string{
			ModelBase:  anthropic.ModelClaude3_5HaikuLatest,
			ModelLarge: anthropic.ModelClaude3_5SonnetLatest,
		},
	}

	return &anthropicProvider{
		settings: settings,
		models:   defaultModels,
		client:   client,
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

func convertToAnthropicMessages(messages []openai.ChatCompletionMessage) ([]anthropic.MessageParam, string) {
	anthropicMessages := make([]anthropic.MessageParam, 0, len(messages))
	var systemPrompt string

	// Extract system message if present
	if len(messages) > 0 && messages[0].Role == "system" {
		systemPrompt = messages[0].Content
	}

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case "assistant":
			// Skip assistant messages as they'll be included in the response
			continue
		case "system":
			// System messages are handled separately
			continue
		}
	}

	return anthropicMessages, systemPrompt
}

func convertToOpenAIResponse(resp *anthropic.Message, model string) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: resp.Content[0].Text,
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
}

func (p *anthropicProvider) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	messages, systemPrompt := convertToAnthropicMessages(req.Messages)
	log.DefaultLogger.Info("model", "model", req.Model)
	model := req.Model.toAnthropic(p.models)

	// Anthropic requires a max tokens value
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}

	// Create Anthropic request
	params := anthropic.MessageNewParams{
		Model:     anthropic.F(model),
		MaxTokens: anthropic.F(int64(req.MaxTokens)),
		Messages:  anthropic.F(messages),
	}

	if systemPrompt != "" {
		params.System = anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(systemPrompt),
		})
	}

	// Make request
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		log.DefaultLogger.Error("error creating anthropic chat completion", "err", err)
		return openai.ChatCompletionResponse{}, err
	}

	return convertToOpenAIResponse(resp, model), nil
}

func (p *anthropicProvider) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	messages, systemPrompt := convertToAnthropicMessages(req.Messages)

	// Anthropic requires a max tokens value
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.F(req.Model.toAnthropic(p.models)),
		MaxTokens: anthropic.F(int64(req.MaxTokens)),
		Messages:  anthropic.F(messages),
	}

	if systemPrompt != "" {
		params.System = anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(systemPrompt),
		})
	}

	stream := p.client.Messages.NewStreaming(ctx, params)
	c := make(chan ChatCompletionStreamResponse)

	go func() {
		defer close(c)

		message := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				log.DefaultLogger.Error("error accumulating message", "err", err)
				c <- ChatCompletionStreamResponse{Error: err}
				return
			}

			switch delta := event.Delta.(type) {
			case anthropic.ContentBlockDeltaEventDelta:
				if delta.Text != "" {
					c <- ChatCompletionStreamResponse{
						ChatCompletionStreamResponse: openai.ChatCompletionStreamResponse{
							ID:      message.ID,
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Model:   string(req.Model),
							Choices: []openai.ChatCompletionStreamChoice{
								{
									Index: 0,
									Delta: openai.ChatCompletionStreamChoiceDelta{
										Content: delta.Text,
									},
									FinishReason: openai.FinishReasonNull,
								},
							},
						},
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			if err != io.EOF {
				log.DefaultLogger.Error("anthropic stream error", "err", err)
				c <- ChatCompletionStreamResponse{Error: err}
			}
		}
	}()

	return c, nil
}

func (p *anthropicProvider) ListAssistants(ctx context.Context, limit *int, order *string, after *string, before *string) (openai.AssistantsList, error) {
	return openai.AssistantsList{}, fmt.Errorf("anthropic does not support assistants")
}
