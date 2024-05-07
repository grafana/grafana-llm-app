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
	ModelBase  = "base"
	ModelLarge = "large"
)

// UnmarshalJSON accepts either OpenAI named models for backwards
// compatability, or the new abstract model names.
func ModelFromString(m string) (Model, error) {
	switch {
	case m == ModelLarge || strings.HasPrefix(m, "gpt-4"):
		return ModelLarge, nil
	case m == ModelBase || strings.HasPrefix(m, "gpt-3.5"):
		return ModelBase, nil
	}
	// TODO: Give users the ability to specify a default model abstraction in settings, and use that here.
	return "", fmt.Errorf("unrecognized model: %s", m)
}

// UnmarshalJSON accepts either OpenAI named models for backwards
// compatability, or the new abstract model names.
func (m *Model) UnmarshalJSON(data []byte) error {
	dataString := string(data)
	switch {
	case dataString == fmt.Sprintf(`"%s"`, ModelLarge) || strings.HasPrefix(dataString, `"gpt-4`):
		*m = ModelLarge
		return nil
	case dataString == fmt.Sprintf(`"%s"`, ModelBase) || strings.HasPrefix(dataString, `"gpt-3.5`):
		*m = ModelBase
		return nil
	}
	// TODO: Give users the ability to specify a default model abstraction in settings, and use that here.
	return fmt.Errorf("unrecognized model: %s", dataString)
}

func (m Model) toOpenAI(modelSettings *ModelSettings) string {
	if modelSettings == nil || len(modelSettings.Models) == 0 {
		switch m {
		case ModelBase:
			return "gpt-3.5-turbo"
		case ModelLarge:
			return "gpt-4-turbo"
		}
		panic(fmt.Sprintf("unrecognized model: %s", m))
	}
	return modelSettings.getModel(m)
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
