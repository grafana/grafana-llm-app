package plugin

import (
	"context"
	"errors"

	"github.com/sashabaranov/go-openai"
)

// Ensure that testProvider implements the LLMProvider interface.
var _ LLMProvider = (*testProvider)(nil)

// testProvider is a test implementation of LLMProvider.
type testProvider struct {
	// ModelsResponse is the response to return from Models.
	ModelsResponse ModelResponse `json:"modelsResponse,omitempty"`

	// ChatCompletionResponse is the response to return from ChatCompletion.
	ChatCompletionResponse openai.ChatCompletionResponse `json:"chatCompletionResponse,omitempty"`

	// ChatCompletionError is an error to return from ChatCompletion.
	// If nil (the default) ChatCompletion will not return an error.
	ChatCompletionError string `json:"chatCompletionError,omitempty"`

	// InitialStreamError is an error to return from ChatCompletionStream.
	// If nil (the default) ChatCompletionStream will not return an error.
	InitialStreamError string `json:"initialStreamError,omitempty"`

	// StreamDeltas is a list of StreamDeltas to return from ChatCompletionStream.
	StreamDeltas []openai.ChatCompletionStreamChoiceDelta `json:"streamDeltas,omitempty"`
	// StreamFinishReason is the reason to finish the stream.
	// Defaults to FinishReasonStop.
	StreamFinishReason openai.FinishReason `json:"streamFinishReason,omitempty"`
	// StreamError is an error to return from ChatCompletionStream after the first delta.
	// If nil (the default) the stream will finish with a stop reason.
	StreamError string `json:"streamError,omitempty"`
}

func defaultTestProvider() testProvider {
	return testProvider{
		ModelsResponse: ModelResponse{
			Data: []ModelInfo{
				{ID: "base"},
				{ID: "large"},
			},
		},

		ChatCompletionResponse: openai.ChatCompletionResponse{
			ID:    "0",
			Model: "tiny",
			Usage: openai.Usage{
				TotalTokens:      10,
				PromptTokens:     5,
				CompletionTokens: 5,
			},
		},
		ChatCompletionError: "",

		InitialStreamError: "",
		StreamDeltas: []openai.ChatCompletionStreamChoiceDelta{
			{Content: "Hello ", Role: openai.ChatMessageRoleAssistant},
			{Content: "there", Role: openai.ChatMessageRoleAssistant},
			{Content: ".", Role: openai.ChatMessageRoleAssistant},
		},
		StreamFinishReason: openai.FinishReasonStop,
		StreamError:        "",
	}
}

func (p *testProvider) Models(context.Context) (ModelResponse, error) {
	return p.ModelsResponse, nil
}

func validateChatCompletionRequest(req ChatCompletionRequest) error {
	if len(req.Messages) == 0 {
		return errors.New("at least one message is required")
	}
	for _, m := range req.Messages {
		if m.Role == "" {
			return errors.New("role is required for each message")
		}
		if m.Content == "" {
			return errors.New("content is required for each message")
		}
	}
	return nil
}

func (p *testProvider) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if p.ChatCompletionError != "" {
		return openai.ChatCompletionResponse{}, errors.New(p.ChatCompletionError)
	}
	if err := validateChatCompletionRequest(req); err != nil {
		return openai.ChatCompletionResponse{}, err
	}
	return p.ChatCompletionResponse, nil
}

func (p *testProvider) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	if err := validateChatCompletionRequest(req); err != nil {
		return nil, err
	}
	if p.InitialStreamError != "" {
		return nil, errors.New(p.InitialStreamError)
	}
	// Use a buffered channel to avoid blocking or requiring a separate goroutine.
	// The buffer size is the number of deltas plus one for the stop reason.
	c := make(chan ChatCompletionStreamResponse, len(p.StreamDeltas)+1)
	defer close(c)
	i := 0
	for _, d := range p.StreamDeltas {

		if i == 1 && p.StreamError != "" {
			c <- ChatCompletionStreamResponse{Error: errors.New(p.StreamError)}
			break
		}

		c <- ChatCompletionStreamResponse{
			ChatCompletionStreamResponse: openai.ChatCompletionStreamResponse{
				Choices: []openai.ChatCompletionStreamChoice{
					{Delta: d},
				},
			},
		}
		i += 1
	}

	// Finish with a stop reason.
	finishReason := openai.FinishReasonStop
	if p.StreamFinishReason != "" {
		finishReason = p.StreamFinishReason
	}
	c <- ChatCompletionStreamResponse{
		ChatCompletionStreamResponse: openai.ChatCompletionStreamResponse{
			Choices: []openai.ChatCompletionStreamChoice{
				{FinishReason: finishReason},
			},
		},
	}
	return c, nil
}
