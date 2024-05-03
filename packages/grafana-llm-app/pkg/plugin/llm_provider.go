package plugin

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var errBadRequest = errors.New("bad request")

type Model string

const (
	ModelDefault      = "default"
	ModelHighAccuracy = "high-accuracy"
)

// UnmarshalJSON accepts either OpenAI named models for backwards
// compatability, or the new abstract model names.
func ModelFromString(m string) (Model, error) {
	switch {
	case strings.HasPrefix(m, "gpt-3.5") || m == ModelDefault:
		return ModelDefault, nil
	case strings.HasPrefix(m, "gpt-4") || m == ModelHighAccuracy:
		return ModelHighAccuracy, nil
	}
	return "", fmt.Errorf("unrecognized model: %s", m)
}

// UnmarshalJSON accepts either OpenAI named models for backwards
// compatability, or the new abstract model names.
func (m *Model) UnmarshalJSON(data []byte) error {
	dataString := string(data)
	switch {
	case dataString == fmt.Sprintf(`"%s"`, ModelDefault) || strings.HasPrefix(dataString, `"gpt-3.5`):
		*m = ModelDefault
		return nil
	case dataString == fmt.Sprintf(`"%s"`, ModelHighAccuracy) || strings.HasPrefix(dataString, `"gpt-4`):
		*m = ModelHighAccuracy
		return nil
	}
	return fmt.Errorf("unrecognized model: %s", dataString)
}

func (m Model) toOpenAI() string {
	switch m {
	case ModelDefault:
		return "gpt-3.5-turbo"
	case ModelHighAccuracy:
		return "gpt-4"
	}
	panic("unknown model: " + m)
}

type Role string

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       Model     `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
}

type ChatCompletionsResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type StreamChatCompletionResponse struct {
	ID      string                       `json:"id"`
	Object  string                       `json:"object"`
	Created int64                        `json:"created"`
	Model   string                       `json:"model"`
	Choices []ChatCompletionStreamChoice `json:"choices"`
}

type ChatCompletionStreamChoice struct {
	Delta        ChoiceDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type ChoiceDelta struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type ModelResponse struct {
	Data []ModelInfo `json:"data"`
}

type ModelInfo struct {
	ID Model `json:"id"`
}

type LLMProvider interface {
	// Models returns a list of models
	Models(context.Context) (ModelResponse, error)
	ChatCompletions(context.Context, ChatCompletionRequest) (ChatCompletionsResponse, error)
	// TODO: Add StreamChatCompletions to this interface so we have one place
	// to implement a new provider.
	// StreamChatCompletions(context.Context, ChatCompletionRequest) (<-chan StreamChatCompletionResponse, error)
}