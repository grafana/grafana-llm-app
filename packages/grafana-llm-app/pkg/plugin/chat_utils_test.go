package plugin

import (
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestForceUserMessage(t *testing.T) {
	tests := []struct {
		name     string
		messages []openai.ChatCompletionMessage
		expected []openai.ChatCompletionMessage
	}{
		{
			name:     "empty messages",
			messages: []openai.ChatCompletionMessage{},
			expected: []openai.ChatCompletionMessage{},
		},
		{
			name: "already has user message",
			messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant.",
				},
				{
					Role:    "user",
					Content: "Hello",
				},
				{
					Role:    "assistant",
					Content: "Hi there!",
				},
			},
			expected: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant.",
				},
				{
					Role:    "user",
					Content: "Hello",
				},
				{
					Role:    "assistant",
					Content: "Hi there!",
				},
			},
		},
		{
			name: "no user message",
			messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant.",
				},
				{
					Role:    "assistant",
					Content: "Hi there!",
				},
			},
			expected: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant.",
				},
				{
					Role:    "user",
					Content: "Hi there!",
				},
			},
		},
		{
			name: "only system message",
			messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant.",
				},
			},
			expected: []openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: "You are a helpful assistant.",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &openai.ChatCompletionRequest{
				Messages: tc.messages,
			}
			ForceUserMessage(req)
			assert.Equal(t, tc.expected, req.Messages)
		})
	}
}

func TestNormalizeTokenLimits(t *testing.T) {
	tests := []struct {
		name                        string
		model                       string
		inputMaxTokens              int
		inputMaxCompletionTokens    int
		expectedMaxTokens           int
		expectedMaxCompletionTokens int
	}{
		{
			name:                        "standard model preserves max tokens",
			model:                       "gpt-4o",
			inputMaxTokens:              1000,
			inputMaxCompletionTokens:    0,
			expectedMaxTokens:           1000,
			expectedMaxCompletionTokens: 0,
		},
		{
			name:                        "o1 model migrates max tokens to max completion tokens",
			model:                       "o1-mini",
			inputMaxTokens:              1500,
			inputMaxCompletionTokens:    0,
			expectedMaxTokens:           0,
			expectedMaxCompletionTokens: 1500,
		},
		{
			name:                        "o3 model migrates max tokens to max completion tokens",
			model:                       "o3-mini",
			inputMaxTokens:              2000,
			inputMaxCompletionTokens:    0,
			expectedMaxTokens:           0,
			expectedMaxCompletionTokens: 2000,
		},
		{
			name:                        "gpt-5 model migrates max tokens to max completion tokens",
			model:                       "gpt-5-mini",
			inputMaxTokens:              3000,
			inputMaxCompletionTokens:    0,
			expectedMaxTokens:           0,
			expectedMaxCompletionTokens: 3000,
		},
		{
			name:                        "o1 model with both set clears max tokens to avoid API error",
			model:                       "o1-preview",
			inputMaxTokens:              1000,
			inputMaxCompletionTokens:    2000,
			expectedMaxTokens:           0,
			expectedMaxCompletionTokens: 2000,
		},
		{
			name:                        "standard model with both set leaves both unchanged",
			model:                       "gpt-4",
			inputMaxTokens:              1000,
			inputMaxCompletionTokens:    2000,
			expectedMaxTokens:           1000,
			expectedMaxCompletionTokens: 2000,
		},
		{
			name:                        "zero tokens remain zero",
			model:                       "o1-preview",
			inputMaxTokens:              0,
			inputMaxCompletionTokens:    0,
			expectedMaxTokens:           0,
			expectedMaxCompletionTokens: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &openai.ChatCompletionRequest{
				Model:               tt.model,
				MaxTokens:           tt.inputMaxTokens,
				MaxCompletionTokens: tt.inputMaxCompletionTokens,
			}
			NormalizeTokenLimits(req)
			assert.Equal(t, tt.expectedMaxTokens, req.MaxTokens)
			assert.Equal(t, tt.expectedMaxCompletionTokens, req.MaxCompletionTokens)
		})
	}
}

func TestRequiresMaxCompletionTokens(t *testing.T) {
	assert.True(t, RequiresMaxCompletionTokens("o1"))
	assert.True(t, RequiresMaxCompletionTokens("o1-mini"))
	assert.True(t, RequiresMaxCompletionTokens("o1-preview"))
	assert.True(t, RequiresMaxCompletionTokens("o3"))
	assert.True(t, RequiresMaxCompletionTokens("o3-mini"))
	assert.True(t, RequiresMaxCompletionTokens("o4"))
	assert.True(t, RequiresMaxCompletionTokens("gpt-5"))
	assert.True(t, RequiresMaxCompletionTokens("gpt-5-4-mini"))

	assert.False(t, RequiresMaxCompletionTokens("gpt-4"))
	assert.False(t, RequiresMaxCompletionTokens("gpt-4o"))
	assert.False(t, RequiresMaxCompletionTokens("gpt-4-turbo"))
	assert.False(t, RequiresMaxCompletionTokens("gpt-3.5-turbo"))
	assert.False(t, RequiresMaxCompletionTokens("claude-3-opus"))
}

func TestIsMaxTokensUnsupportedError(t *testing.T) {
	// Azure Foundry / OpenAI error format
	azureErr := assert.AnError
	assert.False(t, IsMaxTokensUnsupportedError(nil))
	assert.False(t, IsMaxTokensUnsupportedError(azureErr))

	azureMsgErr := &testError{msg: "this model is not supported MaxTokens, please use MaxCompletionTokens"}
	assert.True(t, IsMaxTokensUnsupportedError(azureMsgErr))

	openAIMsgErr := &testError{msg: "Unsupported parameter: 'max_tokens' is not supported with this model. Use 'max_completion_tokens' instead."}
	assert.True(t, IsMaxTokensUnsupportedError(openAIMsgErr))

	unrelatedErr := &testError{msg: "This model's maximum context length is 128000 tokens. However, your messages resulted in 129000 tokens."}
	assert.False(t, IsMaxTokensUnsupportedError(unrelatedErr))

	rateLimitErr := &testError{msg: "Rate limit reached for requests"}
	assert.False(t, IsMaxTokensUnsupportedError(rateLimitErr))
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
