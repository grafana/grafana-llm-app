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
	ModelSmall  = "small"
	ModelMedium = "medium"
	ModelLarge  = "large"
)

var GPT4LargeModels = []string{
	"gpt-4",
	"gpt-4-0613",
	"gpt-4-32k",
	"gpt-4-32k-0613",
}

func contains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// UnmarshalJSON accepts either OpenAI named models for backwards
// compatability, or the new abstract model names.
func ModelFromString(m string) (Model, error) {
	switch {
	case m == ModelLarge || contains(GPT4LargeModels, m):
		return ModelLarge, nil
	case m == ModelMedium || strings.HasPrefix(m, "gpt-4"):
		return ModelMedium, nil
	case m == ModelSmall || strings.HasPrefix(m, "gpt-3.5"):
		return ModelSmall, nil
	}
	// TODO: Give users the ability to specify a default model abstraction in settings, and use that here.
	return "", fmt.Errorf("unrecognized model: %s", m)
}

// UnmarshalJSON accepts either OpenAI named models for backwards
// compatability, or the new abstract model names.
func (m *Model) UnmarshalJSON(data []byte) error {
	dataString := string(data)
	switch {
	case dataString == fmt.Sprintf(`"%s"`, ModelLarge) || contains(GPT4LargeModels, dataString[1:len(dataString)-1]):
		*m = ModelLarge
		return nil
	case dataString == fmt.Sprintf(`"%s"`, ModelMedium) || strings.HasPrefix(dataString, `"gpt-4`):
		*m = ModelMedium
		return nil
	case dataString == fmt.Sprintf(`"%s"`, ModelSmall) || strings.HasPrefix(dataString, `"gpt-3.5`):
		*m = ModelSmall
		return nil
	}
	// TODO: Give users the ability to specify a default model abstraction in settings, and use that here.
	return fmt.Errorf("unrecognized model: %s", dataString)
}

func (m Model) toOpenAI() string {
	// TODO: Add ability to change which model is used for each abstraction in settings.
	switch m {
	case ModelSmall:
		return "gpt-3.5-turbo"
	case ModelMedium:
		return "gpt-4-turbo"
	case ModelLarge:
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
