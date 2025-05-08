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
