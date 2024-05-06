package llmclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestChatCompletions(t *testing.T) {
	ctx := context.Background()
	key := "test"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 page not found"))
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
		}
		req := openai.ChatCompletionRequest{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		response := openai.ChatCompletionResponse{
			ID:    "test",
			Model: "test",
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Role: "system", Content: "test"}},
				{Message: openai.ChatCompletionMessage{Role: "user", Content: "test"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		j, _ := json.Marshal(response)
		w.Write(j)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	// Create a mock OpenAI client
	client := NewOpenAI(server.URL, key)
	// Test case: Chat completions request succeeds
	req := ChatCompletionRequest{
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "/start"},
				{Role: "user", Content: "Hello, how are you?"},
			},
		},
		Model: ModelSmall,
	}
	_, err := client.ChatCompletions(ctx, req)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
