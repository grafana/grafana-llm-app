package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/sashabaranov/go-openai"
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

type ChatCompletionRequest struct {
	openai.ChatCompletionRequest
	Model Model `json:"model"`
}

type ChatCompletionStreamResponse struct {
	openai.ChatCompletionStreamResponse
	// Random padding used to mitigate side channel attacks.
	// See https://blog.cloudflare.com/ai-side-channel-attack-mitigated.
	Padding string `json:"p,omitempty"`
	// Error indicates that an error occurred mid-stream.
	Error error `json:"-"`
}

var unsafeDisablePadding = false

func (r ChatCompletionStreamResponse) MarshalJSON() ([]byte, error) {
	if !unsafeDisablePadding {
		// Define a wrapper type to avoid infinite recursion when calling MarshalJSON below.
		r.Padding = strings.Repeat("p", rand.Int()%35)
	}
	type Wrapper ChatCompletionStreamResponse
	a := (Wrapper)(r)
	return json.Marshal(a)
}

type ModelResponse struct {
	Data []ModelInfo `json:"data"`
}

type ModelInfo struct {
	ID Model `json:"id"`
}

type LLMProvider interface {
	// Models returns a list of models supported by the provider.
	Models(context.Context) (ModelResponse, error)
	// ChatCompletion provides text completion in a chat-like interface.
	ChatCompletion(context.Context, ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	// ChatCompletionStream provides text completion in a chat-like interface with
	// tokens being sent as they are ready.
	ChatCompletionStream(context.Context, ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error)
}
