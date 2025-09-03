package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicProvider_MaxTokensHandling(t *testing.T) {
	tests := []struct {
		name                        string
		inputMaxTokens              int
		inputMaxCompletionTokens    int
		expectedMaxTokens           int
		expectedMaxCompletionTokens int
		description                 string
	}{
		{
			name:                        "both_zero_sets_default_completion_tokens",
			inputMaxTokens:              0,
			inputMaxCompletionTokens:    0,
			expectedMaxTokens:           0,
			expectedMaxCompletionTokens: DefaultMaxCompletionTokens,
			description:                 "When both are zero, MaxCompletionTokens should be set to default",
		},
		{
			name:                        "max_tokens_set_completion_tokens_zero",
			inputMaxTokens:              1000,
			inputMaxCompletionTokens:    0,
			expectedMaxTokens:           1000,
			expectedMaxCompletionTokens: 0,
			description:                 "When MaxTokens is set, MaxCompletionTokens should remain zero",
		},
		{
			name:                        "max_tokens_zero_completion_tokens_set",
			inputMaxTokens:              0,
			inputMaxCompletionTokens:    2000,
			expectedMaxTokens:           0,
			expectedMaxCompletionTokens: 2000,
			description:                 "When MaxCompletionTokens is set, it should remain unchanged",
		},
		{
			name:                        "both_set_remain_unchanged",
			inputMaxTokens:              1500,
			inputMaxCompletionTokens:    2500,
			expectedMaxTokens:           1500,
			expectedMaxCompletionTokens: 2500,
			description:                 "When both are set, both should remain unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("ChatCompletion", func(t *testing.T) {
				testAnthropicMaxTokensHandling(t, tt, false)
			})
			t.Run("ChatCompletionStream", func(t *testing.T) {
				testAnthropicMaxTokensHandling(t, tt, true)
			})
		})
	}
}

func testAnthropicMaxTokensHandling(t *testing.T, tt struct {
	name                        string
	inputMaxTokens              int
	inputMaxCompletionTokens    int
	expectedMaxTokens           int
	expectedMaxCompletionTokens int
	description                 string
}, isStreaming bool) {
	// Create a test server to capture the request
	var capturedRequest openai.ChatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request body to verify the token settings
		err := json.NewDecoder(r.Body).Decode(&capturedRequest)
		require.NoError(t, err)

		// Return a mock response
		if isStreaming {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, `data: {"id":"test","object":"chat.completion.chunk","choices":[{"delta":{"content":"test"}}]}`)
			_, _ = fmt.Fprint(w, "\n\ndata: [DONE]\n\n")
		} else {
			response := openai.ChatCompletionResponse{
				ID: "test-completion",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "test response",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create the anthropic provider with test server
	settings := AnthropicSettings{
		URL:    server.URL,
		apiKey: "test-key",
	}

	provider, err := NewAnthropicProvider(settings, nil)
	require.NoError(t, err)

	// Create the test request
	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "test message",
				},
			},
			MaxTokens:           tt.inputMaxTokens,
			MaxCompletionTokens: tt.inputMaxCompletionTokens,
		},
	}

	ctx := context.Background()

	if isStreaming {
		// Test streaming completion
		respCh, err := provider.ChatCompletionStream(ctx, req)
		require.NoError(t, err)

		// Consume the stream
		for range respCh {
			// Just consume the responses
		}
	} else {
		// Test non-streaming completion
		_, err := provider.ChatCompletion(ctx, req)
		require.NoError(t, err)
	}

	// Verify the captured request has the expected token values
	assert.Equal(t, tt.expectedMaxTokens, capturedRequest.MaxTokens,
		"MaxTokens should be %d, got %d. %s", tt.expectedMaxTokens, capturedRequest.MaxTokens, tt.description)
	assert.Equal(t, tt.expectedMaxCompletionTokens, capturedRequest.MaxCompletionTokens,
		"MaxCompletionTokens should be %d, got %d. %s", tt.expectedMaxCompletionTokens, capturedRequest.MaxCompletionTokens, tt.description)
}

func TestAnthropicProvider_ModelsResponse(t *testing.T) {
	settings := AnthropicSettings{
		URL:    "https://api.anthropic.com",
		apiKey: "test-key",
	}

	provider, err := NewAnthropicProvider(settings, nil)
	require.NoError(t, err)

	models, err := provider.Models(context.Background())
	require.NoError(t, err)

	expectedModels := []ModelInfo{
		{ID: ModelBase},
		{ID: ModelLarge},
	}

	assert.Equal(t, expectedModels, models.Data)
}

func TestAnthropicProvider_ModelMapping(t *testing.T) {
	// Create a test server that captures the model name
	var capturedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		capturedModel = req.Model

		response := openai.ChatCompletionResponse{
			ID: "test-completion",
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "test response",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	settings := AnthropicSettings{
		URL:    server.URL,
		apiKey: "test-key",
	}

	modelSettings := &ModelSettings{
		Mapping: map[Model]string{
			ModelBase:  "claude-3-haiku-20240307",
			ModelLarge: "claude-3-opus-20240229",
		},
	}

	provider, err := NewAnthropicProvider(settings, modelSettings)
	require.NoError(t, err)

	// Test base model mapping
	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "test message",
				},
			},
		},
	}

	_, err = provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "claude-3-haiku-20240307", capturedModel)

	// Test large model mapping
	req.Model = ModelLarge
	_, err = provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "claude-3-opus-20240229", capturedModel)
}

func TestAnthropicProvider_ForceUserMessage(t *testing.T) {
	// Create a test server that captures the messages
	var capturedMessages []openai.ChatCompletionMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		capturedMessages = req.Messages

		response := openai.ChatCompletionResponse{
			ID: "test-completion",
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "test response",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	settings := AnthropicSettings{
		URL:    server.URL,
		apiKey: "test-key",
	}

	provider, err := NewAnthropicProvider(settings, nil)
	require.NoError(t, err)

	// Test with no user messages (last message should be converted to user message)
	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "assistant message",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "another assistant message",
				},
			},
		},
	}

	_, err = provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)

	// Verify that ForceUserMessage was called (last message should be user role)
	require.Len(t, capturedMessages, 2)
	assert.Equal(t, openai.ChatMessageRoleUser, capturedMessages[1].Role)
}

func TestAnthropicProvider_ErrorHandling(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error": {"message": "Invalid request"}}`)
	}))
	defer server.Close()

	settings := AnthropicSettings{
		URL:    server.URL,
		apiKey: "test-key",
	}

	provider, err := NewAnthropicProvider(settings, nil)
	require.NoError(t, err)

	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "test message",
				},
			},
		},
	}

	// Test ChatCompletion error handling
	_, err = provider.ChatCompletion(context.Background(), req)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "400"))

	// Test ChatCompletionStream error handling
	respCh, err := provider.ChatCompletionStream(context.Background(), req)
	if err == nil {
		// If no immediate error, check if we get an error from the channel
		for resp := range respCh {
			if resp.Error != nil {
				err = resp.Error
				break
			}
		}
	}
	require.Error(t, err)
}

func TestNewAnthropicProvider_URLJoinError(t *testing.T) {
	settings := AnthropicSettings{
		URL:    "://invalid-url", // This should cause url.JoinPath to fail
		apiKey: "test-key",
	}

	_, err := NewAnthropicProvider(settings, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "join url")
}
