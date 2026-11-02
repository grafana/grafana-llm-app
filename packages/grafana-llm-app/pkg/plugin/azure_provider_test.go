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

func TestAzureProviderAPIVersion(t *testing.T) {
	defaultVersion := openai.DefaultAzureConfig("", "").APIVersion
	for _, stream := range []bool{false, true} {
		pathName := "non-streaming"
		if stream {
			pathName = "streaming"
		}
		for _, tt := range []struct {
			name            string
			configured      string
			expectedVersion string
		}{
			{name: "uses SDK default when unset", expectedVersion: defaultVersion},
			{name: "uses configured version", configured: "2024-10-21", expectedVersion: "2024-10-21"},
		} {
			t.Run(pathName+"/"+tt.name, func(t *testing.T) {
				var receivedVersion string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					receivedVersion = r.URL.Query().Get("api-version")
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
						return
					}
					_ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{ID: "test"})
				}))
				defer server.Close()

				provider, err := NewAzureProvider(OpenAISettings{
					URL:          server.URL,
					APIVersion:   tt.configured,
					apiKey:       "test-key",
					AzureMapping: [][]string{{ModelBase, "deployment"}},
				}, ModelBase)
				require.NoError(t, err)

				req := ChatCompletionRequest{
					Model: ModelBase,
					ChatCompletionRequest: openai.ChatCompletionRequest{
						Messages: []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "test"}},
					},
				}
				if stream {
					responses, err := provider.ChatCompletionStream(context.Background(), req)
					require.NoError(t, err)
					for range responses {
					}
				} else {
					_, err := provider.ChatCompletion(context.Background(), req)
					require.NoError(t, err)
				}

				assert.Equal(t, tt.expectedVersion, receivedVersion)
			})
		}
	}
}

func TestAzureProvider_TokenNormalization(t *testing.T) {
	var capturedReq openai.ChatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		_ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{
			ID: "test-success",
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "response"}},
			},
		})
	}))
	defer server.Close()

	provider, err := NewAzureProvider(OpenAISettings{
		URL:          server.URL,
		apiKey:       "test-key",
		AzureMapping: [][]string{{ModelBase, "o1-mini-deployment"}},
	}, ModelBase)
	require.NoError(t, err)

	req := ChatCompletionRequest{
		Model: ModelBase,
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages:  []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "hi"}},
			MaxTokens: 1200,
		},
	}

	_, err = provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, 0, capturedReq.MaxTokens)
	assert.Equal(t, 1200, capturedReq.MaxCompletionTokens)
}

func TestAzureProvider_MaxTokensAutoFallback(t *testing.T) {
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
					_, _ = fmt.Fprint(w, `{"error": {"message": "this model is not supported MaxTokens, please use MaxCompletionTokens"}}`)
					return
				}

				if stream {
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = fmt.Fprint(w, "data: {\"id\":\"test-stream\",\"choices\":[{\"delta\":{\"content\":\"streamed content\"}}]}\n\n")
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

			provider, err := NewAzureProvider(OpenAISettings{
				URL:          server.URL,
				apiKey:       "test-key",
				AzureMapping: [][]string{{ModelBase, "custom-foundry-deployment"}},
			}, ModelBase)
			require.NoError(t, err)

			req := ChatCompletionRequest{
				Model: ModelBase,
				ChatCompletionRequest: openai.ChatCompletionRequest{
					Messages:  []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "hello"}},
					MaxTokens: 1500,
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

			assert.Equal(t, 2, attempts, "expected provider to retry once after receiving MaxTokens error")
			require.Len(t, capturedRequests, 2)
			assert.Equal(t, 1500, capturedRequests[0].MaxTokens)
			assert.Equal(t, 0, capturedRequests[0].MaxCompletionTokens)
			assert.Equal(t, 0, capturedRequests[1].MaxTokens)
			assert.Equal(t, 1500, capturedRequests[1].MaxCompletionTokens)
		})
	}
}
