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

func TestOpenAIProviders_MaxTokensOutboundJSON(t *testing.T) {
	tests := []struct {
		name                        string
		model                       string
		maxTokens                   int
		maxCompletionTokens         int
		expectedMaxTokens           int
		expectedMaxCompletionTokens int
	}{
		{
			name:                        "GPT-5 converts max_tokens",
			model:                       "gpt-5.4-mini",
			maxTokens:                   100,
			expectedMaxCompletionTokens: 100,
		},
		{
			name:                        "GPT-5 preserves caller-supplied max_completion_tokens",
			model:                       "gpt-5.4-mini",
			maxTokens:                   100,
			maxCompletionTokens:         200,
			expectedMaxCompletionTokens: 200,
		},
		{
			name:              "older model preserves max_tokens",
			model:             "gpt-4.1-mini",
			maxTokens:         100,
			expectedMaxTokens: 100,
		},
	}

	providers := []struct {
		name string
		new  func(t *testing.T, serverURL, model string) LLMProvider
	}{
		{
			name: "OpenAI-compatible",
			new: func(t *testing.T, serverURL, model string) LLMProvider {
				t.Helper()
				apiPath := "/v1"
				provider, err := NewOpenAIProvider(OpenAISettings{
					URL:     serverURL,
					APIPath: &apiPath,
					apiKey:  "test-key",
				}, &ModelSettings{
					Default: ModelBase,
					Mapping: map[Model]string{ModelBase: model},
				})
				require.NoError(t, err)
				return provider
			},
		},
		{
			name: "Azure",
			new: func(t *testing.T, serverURL, model string) LLMProvider {
				t.Helper()
				provider, err := NewAzureProvider(OpenAISettings{
					URL:          serverURL,
					apiKey:       "test-key",
					AzureMapping: [][]string{{ModelBase, model}},
				}, ModelBase)
				require.NoError(t, err)
				return provider
			},
		},
	}

	for _, providerTest := range providers {
		for _, stream := range []bool{false, true} {
			pathName := "non-streaming"
			if stream {
				pathName = "streaming"
			}
			for _, tt := range tests {
				t.Run(providerTest.name+"/"+pathName+"/"+tt.name, func(t *testing.T) {
					var outbound map[string]json.RawMessage
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if err := json.NewDecoder(r.Body).Decode(&outbound); err != nil {
							t.Errorf("decode outbound request: %v", err)
							http.Error(w, "invalid request", http.StatusBadRequest)
							return
						}
						if stream {
							w.Header().Set("Content-Type", "text/event-stream")
							_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
							return
						}
						_ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{ID: "test"})
					}))
					defer server.Close()

					provider := providerTest.new(t, server.URL, tt.model)
					req := ChatCompletionRequest{
						Model: ModelBase,
						ChatCompletionRequest: openai.ChatCompletionRequest{
							Messages: []openai.ChatCompletionMessage{{
								Role:    openai.ChatMessageRoleUser,
								Content: "test",
							}},
							MaxTokens:           tt.maxTokens,
							MaxCompletionTokens: tt.maxCompletionTokens,
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

					assertJSONInteger(t, outbound, "max_tokens", tt.expectedMaxTokens)
					assertJSONInteger(t, outbound, "max_completion_tokens", tt.expectedMaxCompletionTokens)
				})
			}
		}
	}
}

func assertJSONInteger(t *testing.T, body map[string]json.RawMessage, field string, expected int) {
	t.Helper()
	value, ok := body[field]
	if expected == 0 {
		assert.False(t, ok, "%s should be omitted from outbound JSON", field)
		return
	}
	require.True(t, ok, "%s should be present in outbound JSON", field)
	assert.JSONEq(t, fmt.Sprintf("%d", expected), string(value))
}
