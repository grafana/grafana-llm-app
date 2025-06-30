package plugin

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestModelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Model
		wantErr  bool
	}{
		{
			input:    "base",
			expected: ModelBase,
			wantErr:  false,
		},
		{
			input:    "large",
			expected: ModelLarge,
			wantErr:  false,
		},

		// unknown models
		{
			input:    "invalid_model",
			expected: "",
			wantErr:  true,
		},
		{
			input:    "",
			expected: "",
			wantErr:  true,
		},

		// backwards-compatibility
		{
			input:    "gpt-3.5-turbo",
			expected: ModelBase,
			wantErr:  false,
		},
		{
			input:    "gpt-3.5-turbo-0125",
			expected: ModelBase,
			wantErr:  false,
		},
		{
			input:    "gpt-4o-mini",
			expected: ModelBase,
			wantErr:  false,
		},
		{
			input:    "gpt-4o-mini-2024-07-18",
			expected: ModelBase,
			wantErr:  false,
		},
		{
			input:    "gpt-4-turbo",
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    "gpt-4-turbo-2024-04-09",
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    "gpt-4",
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    "gpt-4o",
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    "gpt-4-32k-0613",
			expected: ModelLarge,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ModelFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModelFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ModelFromString() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestModelUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    []byte
		expected Model
		wantErr  bool
	}{
		{
			input:    []byte(`"base"`),
			expected: ModelBase,
			wantErr:  false,
		},
		{
			input:    []byte(`"large"`),
			expected: ModelLarge,
			wantErr:  false,
		},

		// unknown models
		{
			input:    []byte(`"invalid_model"`),
			expected: "",
			wantErr:  true,
		},
		{
			input:    []byte(`""`),
			expected: "",
			wantErr:  true,
		},
		{
			input:    []byte(`null`),
			expected: "",
			wantErr:  true,
		},

		// backwards-compatibility
		{
			input:    []byte(`"gpt-3.5-turbo"`),
			expected: ModelBase,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-3.5-turbo-0125"`),
			expected: ModelBase,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-4-turbo"`),
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-4-turbo-2024-04-09"`),
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-4"`),
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-4-32k-0613"`),
			expected: ModelLarge,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			var m Model
			err := m.UnmarshalJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if m != tt.expected {
				t.Errorf("UnmarshalJSON() = %v, expected %v", m, tt.expected)
			}
		})
	}
}

func TestChatCompletionRequestUnmarshalJSON(t *testing.T) {
	for _, tt := range []struct {
		input    []byte
		expected ChatCompletionRequest
	}{
		{
			input: []byte(`{"model":"base"}`),
			expected: ChatCompletionRequest{
				Model: ModelBase,
				ChatCompletionRequest: openai.ChatCompletionRequest{
					Temperature: 0,
				},
			},
		},
		{
			input: []byte(`{"model":"base", "temperature":0.5}`),
			expected: ChatCompletionRequest{
				Model: ModelBase,
				ChatCompletionRequest: openai.ChatCompletionRequest{
					Temperature: 0.5,
				},
			},
		},
		{
			input: []byte(`{"model":"base", "temperature":0}`),
			expected: ChatCompletionRequest{
				Model: ModelBase,
				ChatCompletionRequest: openai.ChatCompletionRequest{
					Temperature: math.SmallestNonzeroFloat32,
				},
			},
		},
	} {
		t.Run(string(tt.input), func(t *testing.T) {
			var req ChatCompletionRequest
			err := json.Unmarshal(tt.input, &req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, req)
		})
	}
}

func TestChatCompletionStreamResponseMarshalJSON(t *testing.T) {
	resp := ChatCompletionStreamResponse{
		ChatCompletionStreamResponse: openai.ChatCompletionStreamResponse{
			ID: "123",
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("error marshaling ChatCompletionStreamResponse: %s", err)
	}
	var got map[string]any
	err = json.Unmarshal(b, &got)
	if err != nil {
		t.Errorf("error unmarshaling ChatCompletionStreamResponse: %s", err)
	}
	_, ok := got["p"]
	if !ok {
		t.Errorf("no padding found in ChatCompletionStreamResponse")
	}
	if got["id"] != "123" {
		t.Errorf("id doesn't match")
	}
}

func TestModelToAnthropic(t *testing.T) {
	for _, tt := range []struct {
		input    Model
		settings *ModelSettings
		expected string
	}{
		{
			input:    ModelBase,
			settings: nil,
			expected: string(anthropic.ModelClaude4Sonnet20250514),
		},
		{
			input:    ModelLarge,
			settings: nil,
			expected: string(anthropic.ModelClaude4Sonnet20250514),
		},
		{
			input: ModelBase,
			settings: &ModelSettings{
				Mapping: map[Model]string{
					ModelBase:  string(anthropic.ModelClaude4Sonnet20250514),
					ModelLarge: string(anthropic.ModelClaude4Sonnet20250514),
				},
			},
			expected: string(anthropic.ModelClaude4Sonnet20250514),
		},
		{
			input: ModelLarge,
			settings: &ModelSettings{
				Mapping: map[Model]string{
					ModelBase:  string(anthropic.ModelClaude4Sonnet20250514),
					ModelLarge: string(anthropic.ModelClaude4Sonnet20250514),
				},
			},
			expected: string(anthropic.ModelClaude4Sonnet20250514),
		},
		{
			input: ModelLarge,
			settings: &ModelSettings{
				Mapping: map[Model]string{
					ModelLarge: string(anthropic.ModelClaude4Sonnet20250514),
				},
			},
			expected: string(anthropic.ModelClaude4Sonnet20250514),
		},
		{
			input: ModelLarge,
			settings: &ModelSettings{
				Mapping: map[Model]string{
					ModelBase: string(anthropic.ModelClaude4Sonnet20250514),
				},
			},
			expected: string(anthropic.ModelClaude4Sonnet20250514),
		},
	} {
		t.Run(string(tt.input), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.toAnthropic(tt.settings))
		})
	}
}

func TestModelToOpenAI(t *testing.T) {
	for _, tt := range []struct {
		input    Model
		settings *ModelSettings
		expected string
	}{
		{
			input:    ModelBase,
			settings: nil,
			expected: openai.GPT4Dot1Mini,
		},
		{
			input:    ModelLarge,
			settings: nil,
			expected: openai.GPT4Dot1,
		},
		{
			input: ModelBase,
			settings: &ModelSettings{
				Mapping: map[Model]string{
					ModelBase:  openai.GPT4Dot1Mini,
					ModelLarge: openai.GPT4Dot1,
				},
			},
			expected: openai.GPT4Dot1Mini,
		},
		{
			input: ModelLarge,
			settings: &ModelSettings{
				Mapping: map[Model]string{
					ModelBase:  openai.GPT4Dot1Mini,
					ModelLarge: openai.GPT4Dot1,
				},
			},
			expected: openai.GPT4Dot1,
		},
		{
			input: ModelLarge,
			settings: &ModelSettings{
				Mapping: map[Model]string{
					ModelBase: openai.GPT4Dot1Mini,
				},
			},
			// Note: partial mapping provided, so we use the default model.
			expected: openai.GPT4Dot1,
		},
		{
			input: ModelLarge,
			settings: &ModelSettings{
				Mapping: map[Model]string{
					ModelLarge: openai.GPT4Dot1,
				},
			},
			expected: openai.GPT4Dot1,
		},
	} {
		t.Run(string(tt.input), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.toOpenAI(tt.settings))
		})
	}
}
