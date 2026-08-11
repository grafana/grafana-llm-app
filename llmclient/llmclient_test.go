package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestChatCompletions(t *testing.T) {
	ctx := context.Background()
	key := "test"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/plugins/grafana-llm-app/resources/llm/v1/chat/completions" {
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
	// Create a mock LLM provider client
	client := NewLLMProvider(server.URL, key)
	// Test case: Chat completions request succeeds
	req := ChatCompletionRequest{
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "/start"},
				{Role: "user", Content: "Hello, how are you?"},
			},
		},
		Model: ModelBase,
	}
	_, err := client.ChatCompletions(ctx, req)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestChatCompletionsStream(t *testing.T) {
	ctx := context.Background()
	key := "test"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/plugins/grafana-llm-app/resources/llm/v1/chat/completions" {
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
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, choice := range []openai.ChatCompletionStreamChoice{
			{Delta: openai.ChatCompletionStreamChoiceDelta{Content: "hello"}},
			{Delta: openai.ChatCompletionStreamChoiceDelta{Content: " there"}},
			{FinishReason: openai.FinishReasonStop},
		} {
			response := openai.ChatCompletionStreamResponse{
				ID:      "test",
				Model:   "test",
				Choices: []openai.ChatCompletionStreamChoice{choice},
			}
			j, _ := json.Marshal(response)
			w.Write([]byte("data: " + string(j) + "\n\n"))
		}
		w.Write([]byte("data: [DONE]\n\n"))
	})

	server := httptest.NewServer(handler)
	defer server.Close()
	// Create a mock LLM provider client
	client := NewLLMProvider(server.URL, key)
	// Test case: Chat completions request succeeds
	req := ChatCompletionRequest{
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "/start"},
				{Role: "user", Content: "Hello, how are you?"},
			},
			Stream: true,
		},
		Model: ModelBase,
	}
	stream, err := client.ChatCompletionsStream(ctx, req)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	content := ""
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Errorf("expected no error in stream, got %v", err)
		}
		if resp.Choices[0].FinishReason == openai.FinishReasonStop {
			break
		}
		content += resp.Choices[0].Delta.Content
	}
	if content != "hello there" {
		t.Errorf("expected streamed content to be 'hello there', got '%s'", content)
	}
}

func TestEnabled_LimitsResponseBodyRead(t *testing.T) {
	ctx := context.Background()
	key := "test"
	const oversized = 5 * 1024 * 1024 // 5MiB, well over the intended read cap
	filler := bytes.Repeat([]byte("x"), oversized)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/plugins/grafana-llm-app/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Not valid JSON in either the new or old health response shape, so Enabled
		// is expected to return an error regardless of how much of it gets read.
		w.WriteHeader(http.StatusOK)
		w.Write(filler)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	var bytesRead int64
	transport := &countingTransport{base: http.DefaultTransport, read: &bytesRead}
	client := NewLLMProviderWithClient(server.URL, key, &http.Client{Transport: transport})

	_, err := client.Enabled(ctx)
	if err == nil {
		t.Fatal("expected an error unmarshalling the oversized non-JSON health response, got nil")
	}
	// The read must be capped well below the full response size; otherwise a slow or
	// malicious upstream health response can force this client to buffer it all in memory.
	const cap = 2 * 1024 * 1024 // generous slack above the 1MiB limit
	if bytesRead > cap {
		t.Fatalf("Enabled read %d bytes of the response body, want capped near 1MiB (< %d)", bytesRead, cap)
	}
}

type countingTransport struct {
	base http.RoundTripper
	read *int64
}

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	resp.Body = &countingReadCloser{ReadCloser: resp.Body, read: t.read}
	return resp, nil
}

type countingReadCloser struct {
	io.ReadCloser
	read *int64
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	atomic.AddInt64(c.read, int64(n))
	return n, err
}
