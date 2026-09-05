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

func TestAnthropicProvider_MaxCompletionTokensHandling(t *testing.T) {
	tests := []struct {
		name                        string
		inputMaxCompletionTokens    int
		expectedMaxCompletionTokens int
		description                 string
	}{
		{
			name:                        "zero_sets_default",
			inputMaxCompletionTokens:    0,
			expectedMaxCompletionTokens: DefaultMaxCompletionTokens,
			description:                 "When zero, MaxCompletionTokens should be set to default",
		},
		{
			name:                        "nonzero_remains_unchanged",
			inputMaxCompletionTokens:    2000,
			expectedMaxCompletionTokens: 2000,
			description:                 "When set, MaxCompletionTokens should remain unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("ChatCompletion", func(t *testing.T) {
				testAnthropicMaxCompletionTokensHandling(t, tt, false)
			})
			t.Run("ChatCompletionStream", func(t *testing.T) {
				testAnthropicMaxCompletionTokensHandling(t, tt, true)
			})
		})
	}
}

func testAnthropicMaxCompletionTokensHandling(t *testing.T, tt struct {
	name                        string
	inputMaxCompletionTokens    int
	expectedMaxCompletionTokens int
	description                 string
}, isStreaming bool) {
	var capturedRequest openai.ChatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&capturedRequest)
		require.NoError(t, err)

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
			MaxCompletionTokens: tt.inputMaxCompletionTokens,
		},
	}

	ctx := context.Background()

	if isStreaming {
		respCh, err := provider.ChatCompletionStream(ctx, req)
		require.NoError(t, err)
		for range respCh {
		}
	} else {
		_, err := provider.ChatCompletion(ctx, req)
		require.NoError(t, err)
	}

	assert.Equal(t, tt.expectedMaxCompletionTokens, capturedRequest.MaxCompletionTokens,
		"MaxCompletionTokens should be %d, got %d. %s", tt.expectedMaxCompletionTokens, capturedRequest.MaxCompletionTokens, tt.description)
}

func TestAnthropicProvider_ModelMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		assert.Equal(t, "claude-3-opus-20240229", req.Model)

		response := openai.ChatCompletionResponse{
			ID: "test-completion",
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "test"}},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	settings := AnthropicSettings{
		URL:    server.URL,
		apiKey: "test-key",
	}
	models := &ModelSettings{Mapping: map[Model]string{ModelBase: "claude-3-opus-20240229"}}
	provider, err := NewAnthropicProvider(settings, models)
	require.NoError(t, err)

	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "test"}},
		},
	}

	_, err = provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
}

func TestAnthropicProvider_ForceUserMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		require.NotEmpty(t, req.Messages)
		lastMsg := req.Messages[len(req.Messages)-1]
		assert.Equal(t, openai.ChatMessageRoleUser, lastMsg.Role)

		response := openai.ChatCompletionResponse{
			ID: "test-completion",
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "test"}},
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

	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant."},
			},
		},
	}

	_, err = provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
}

func TestAnthropicProvider_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error": {"message": "Invalid request", "type": "invalid_request_error"}}`)
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
				{Role: openai.ChatMessageRoleUser, Content: "test"},
			},
		},
	}

	_, err = provider.ChatCompletion(context.Background(), req)
	assert.Error(t, err)

	_, err = provider.ChatCompletionStream(context.Background(), req)
	assert.Error(t, err)
}

func TestNewAnthropicProvider_URLJoinError(t *testing.T) {
	settings := AnthropicSettings{
		URL:    "://invalid-url",
		apiKey: "test-key",
	}
	provider, err := NewAnthropicProvider(settings, nil)
	assert.Nil(t, provider)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "join url"))
}
