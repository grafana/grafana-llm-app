package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_TokenNormalization(t *testing.T) {
	var capturedReq openai.ChatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		_ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{
			ID: "test-openai",
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "reply"}},
			},
		})
	}))
	defer server.Close()

	apiPath := ""
	models := &ModelSettings{
		Mapping: map[Model]string{
			ModelBase:  "o1-mini",
			ModelLarge: "gpt-4o",
		},
	}
	provider, err := NewOpenAIProvider(OpenAISettings{
		URL:     server.URL,
		APIPath: &apiPath,
		apiKey:  "test-key",
	}, models)
	require.NoError(t, err)

	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages:  []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "prompt"}},
			MaxTokens: 800,
		},
	}

	_, err = provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, 0, capturedReq.MaxTokens)
	assert.Equal(t, 800, capturedReq.MaxCompletionTokens)
}

func TestOpenAIProvider_MaxTokensAutoFallback(t *testing.T) {
	for _, stream := range []bool{false, true} {
		pathName := "non-streaming"
		if stream {
			pathName = "streaming"
		}
		t.Run(pathName, func(t *testing.T) {
			attempts := 0
			var capturedRequests []openai.ChatCompletionRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req openai.ChatCompletionRequest
				_ = json.NewDecoder(r.Body).Decode(&req)
				capturedRequests = append(capturedRequests, req)
				attempts++

				if attempts == 1 {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = fmt.Fprint(w, `{"error": {"message": "Unsupported parameter: 'max_tokens' is not supported with this model. Use 'max_completion_tokens' instead."}}`)
					return
				}

				if stream {
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = fmt.Fprint(w, "data: {\"id\":\"test-stream\",\"choices\":[{\"delta\":{\"content\":\"stream content\"}}]}\n\n")
					_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
					return
				}

				_ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{
					ID: "test-success",
					Choices: []openai.ChatCompletionChoice{
						{Message: openai.ChatCompletionMessage{Content: "success content"}},
					},
				})
			}))
			defer server.Close()

			apiPath := ""
			models := &ModelSettings{
				Mapping: map[Model]string{
					ModelBase: "custom-model",
				},
			}
			provider, err := NewOpenAIProvider(OpenAISettings{
				URL:     server.URL,
				APIPath: &apiPath,
				apiKey:  "test-key",
			}, models)
			require.NoError(t, err)

			req := ChatCompletionRequest{
				Model: ModelBase,
				ChatCompletionRequest: openai.ChatCompletionRequest{
					Messages:  []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "hello"}},
					MaxTokens: 500,
				},
			}

			if stream {
				respChan, err := provider.ChatCompletionStream(context.Background(), req)
				require.NoError(t, err)
				var items []ChatCompletionStreamResponse
				for resp := range respChan {
					items = append(items, resp)
				}
				require.NotEmpty(t, items)
				assert.NoError(t, items[0].Error)
			} else {
				resp, err := provider.ChatCompletion(context.Background(), req)
				require.NoError(t, err)
				assert.Equal(t, "test-success", resp.ID)
			}

			assert.Equal(t, 2, attempts, "expected provider to retry once on MaxTokens unsupported error")
			require.Len(t, capturedRequests, 2)
			assert.Equal(t, 500, capturedRequests[0].MaxTokens)
			assert.Equal(t, 0, capturedRequests[0].MaxCompletionTokens)
			assert.Equal(t, 0, capturedRequests[1].MaxTokens)
			assert.Equal(t, 500, capturedRequests[1].MaxCompletionTokens)
		})
	}
}

func TestOpenAIProvider_OtherErrorsNotRetried(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprint(w, `{"error": {"message": "Rate limit exceeded"}}`)
	}))
	defer server.Close()

	apiPath := ""
	provider, err := NewOpenAIProvider(OpenAISettings{
		URL:     server.URL,
		APIPath: &apiPath,
		apiKey:  "test-key",
	}, nil)
	require.NoError(t, err)

	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages:  []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "test"}},
			MaxTokens: 100,
		},
	}

	_, err = provider.ChatCompletion(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, 1, attempts, "rate limit error should not be retried")
}
