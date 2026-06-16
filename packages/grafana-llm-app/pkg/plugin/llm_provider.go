package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	case m == ModelLarge || (strings.HasPrefix(m, "gpt-4") && !strings.Contains(m, "-mini")):
		return ModelLarge, nil
	case m == ModelBase || strings.HasPrefix(m, "gpt-3.5") || strings.Contains(m, "-mini"):
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
	return m.toProvider(ProviderTypeOpenAI, modelSettings)
}

func (m Model) toAnthropic(modelSettings *ModelSettings) string {
	return m.toProvider(ProviderTypeAnthropic, modelSettings)
}

func (m Model) toProvider(provider ProviderType, modelSettings *ModelSettings) string {
	defaults := defaultModelSettings(provider)
	// First check for nil settings, in which case we should use the defaults.
	if modelSettings == nil {
		modelSettings = defaults
	}
	// Next make sure that all of the model IDs have a corresponding name in the
	// mapping, falling back to the default name if not.
	for id, name := range defaults.Mapping {
		if modelSettings.Mapping[id] == "" {
			modelSettings.Mapping[id] = name
		}
	}
	if modelSettings.Default == "" {
		modelSettings.Default = defaults.Default
	}
	return modelSettings.getModel(m)
}

type ChatCompletionRequest struct {
	openai.ChatCompletionRequest
	Model Model `json:"model"`
}

// UnmarshalJSON implements json.Unmarshaler.
// We have a custom implementation here to check whether temperature is being
// explicitly set to `0` in the incoming request, because the `openai.ChatCompletionRequest`
// struct has `omitempty` on the Temperature field and would omit it when marshaling.
// If there is an explicit 0 value in the request, we set it to `math.SmallestNonzeroFloat32`,
// a workaround mentioned in https://github.com/sashabaranov/go-openai/issues/9#issuecomment-894845206.
func (c *ChatCompletionRequest) UnmarshalJSON(data []byte) error {
	// Create a wrapper type alias to avoid recursion, otherwise the
	// subsequent call to UnmarshalJSON would call this method forever.
	type Alias ChatCompletionRequest
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	// Also unmarshal to a map to check if temperature is being set explicitly in the request.
	r := map[string]any{}
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	if t, ok := r["temperature"].(float64); ok && t == 0 {
		a.Temperature = math.SmallestNonzeroFloat32
	}
	*c = ChatCompletionRequest(a)
	return nil
}

type ChatCompletionStreamResponse struct {
	openai.ChatCompletionStreamResponse
	// Random padding used to mitigate side channel attacks.
	// See https://blog.cloudflare.com/ai-side-channel-attack-mitigated.
	Padding string `json:"p,omitempty"`
	// Error indicates that an error occurred mid-stream.
	Error error `json:"-"`

	// ignorePadding is a flag to ignore padding in responses.
	// It should only ever be set in tests.
	ignorePadding bool
}

func (r ChatCompletionStreamResponse) MarshalJSON() ([]byte, error) {
	if !r.ignorePadding {
		// Define a wrapper type to avoid infinite recursion when calling MarshalJSON below.
		r.Padding = strings.Repeat("p", rand.Int()%35+1)
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
