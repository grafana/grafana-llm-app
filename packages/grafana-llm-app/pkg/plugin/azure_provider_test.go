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
