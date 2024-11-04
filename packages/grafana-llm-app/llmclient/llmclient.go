// package llmclient provides a client for the Grafana LLM app.
// It is used to communicate with LLM providers via the Grafana LLM app
// using the configuration stored in the app to handle authentication.
package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

const (
	appPrefix          = "/api/plugins/grafana-llm-app"
	appResourcesPrefix = appPrefix + "/resources"
)

// Model is an abstraction over the different models available in different providers
type Model string

const (
	// ModelBase is the base model, for efficient and high-throughput tasks. OpenAI default: gpt-4o-mini
	ModelBase = "base"
	// ModelLarge is the large model, for more advanced tasks with longer context windows. OpenAI default: gpt-4o
	ModelLarge = "large"
)

// ChatCompletionRequest is a request for chat completions using an abstract model.
type ChatCompletionRequest struct {
	openai.ChatCompletionRequest
	Model Model `json:"model"`
}

// AssistantRequest is a request for creating an assistant using an abstract model.
type AssistantRequest struct {
	openai.AssistantRequest
	Model Model `json:"model"`
}

// OpenAI is an interface for talking to OpenAI via the Grafana LLM app.
// Requests made using this interface will be routed to the OpenAI backend
// configured in the Grafana LLM app's settings, with authentication handled
// by the LLM app.
type OpenAI interface {
	// Enabled returns true if the Grafana LLM app has been configured for use
	// with OpenAI.
	Enabled(ctx context.Context) (bool, error)
	// ChatCompletions makes a request to the OpenAI Chat Completion API.
	ChatCompletions(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	// ChatCompletionsStream makes a streaming request to the OpenAI Chat Completion API.
	ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest) (*openai.ChatCompletionStream, error)
	// CreateAssistant creates an assistant using the given request.
	CreateAssistant(ctx context.Context, req AssistantRequest) (openai.Assistant, error)
	// RetrieveAssistant retrieves an assistant by ID.
	RetrieveAssistant(ctx context.Context, assistantID string) (openai.Assistant, error)
	// ListAssistants lists assistants.
	ListAssistants(ctx context.Context, limit *int, order *string, after *string, before *string) (openai.AssistantsList, error)
	// DeleteAssistant deletes an assistant by ID.
	DeleteAssistant(ctx context.Context, assistantID string) (openai.AssistantDeleteResponse, error)
	// CreateThread creates a new thread.
	CreateThread(ctx context.Context, req openai.ThreadRequest) (openai.Thread, error)
	// RetrieveThread retrieves a thread by ID.
	RetrieveThread(ctx context.Context, threadID string) (openai.Thread, error)
	// DeleteThread deletes a thread by ID.
	DeleteThread(ctx context.Context, threadID string) (openai.ThreadDeleteResponse, error)
	// CreateMessage creates a new message in a thread.
	CreateMessage(ctx context.Context, threadID string, request openai.MessageRequest) (msg openai.Message, err error)
	// ListMessages lists messages in a thread.
	ListMessages(ctx context.Context, threadID string, limit *int, order *string, after *string, before *string) (openai.MessagesList, error)
	// RetrieveMessage retrieves a message in a thread.
	RetrieveMessage(ctx context.Context, threadID string, messageID string) (msg openai.Message, err error)
	// DeleteMessage deletes a message in a thread.
	DeleteMessage(ctx context.Context, threadID string, messageID string) (msg openai.MessageDeletionStatus, err error)
	// CreateRun creates a new run in a thread.
	CreateRun(ctx context.Context, threadID string, request openai.RunRequest) (run openai.Run, err error)
	// RetrieveRun retrieves a run in a thread.
	RetrieveRun(ctx context.Context, threadID string, runID string) (run openai.Run, err error)
	// CancelRun cancels a run in a thread.
	CancelRun(ctx context.Context, threadID string, runID string) (run openai.Run, err error)
	// SubmitToolOutputs submits tool outputs for a run in a thread.
	SubmitToolOutputs(ctx context.Context, threadID string, runID string, request openai.SubmitToolOutputsRequest) (response openai.Run, err error)
	// ListMessage lists messages in a thread.
	ListMessage(ctx context.Context, threadID string, limit *int, order *string, after *string, before *string) (openai.MessagesList, error)
}

type openAI struct {
	httpClient *http.Client
	client     *openai.Client

	grafanaURL, grafanaAPIKey string
}

// NewOpenAI creates a new OpenAI client talking to the Grafana LLM app installed
// on the given Grafana instance.
func NewOpenAI(grafanaURL, grafanaAPIKey string) OpenAI {
	httpClient := &http.Client{}
	return NewOpenAIWithClient(grafanaURL, grafanaAPIKey, httpClient)
}

// NewOpenAIWithClient creates a new OpenAI client talking to the Grafana LLM app installed
// on the given Grafana instance, using the given HTTP client.
func NewOpenAIWithClient(grafanaURL, grafanaAPIKey string, httpClient *http.Client) OpenAI {
	grafanaURL = strings.TrimRight(grafanaURL, "/")
	url := grafanaURL + appResourcesPrefix + "/openai/v1"
	cfg := openai.DefaultConfig(grafanaAPIKey)
	cfg.BaseURL = url
	cfg.HTTPClient = httpClient
	client := openai.NewClientWithConfig(cfg)
	return &openAI{
		httpClient:    httpClient,
		client:        client,
		grafanaURL:    grafanaURL,
		grafanaAPIKey: grafanaAPIKey,
	}
}

type openAIModelHealth struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type openAIHealthDetails struct {
	Configured bool                        `json:"configured"`
	OK         bool                        `json:"ok"`
	Error      string                      `json:"error,omitempty"`
	Models     map[Model]openAIModelHealth `json:"models"`
	Assistant  openAIModelHealth           `json:"assistant"`
}

type vectorHealthDetails struct {
	Enabled bool   `json:"enabled"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

type healthCheckDetails struct {
	OpenAI  openAIHealthDetails `json:"openAI"`
	Vector  vectorHealthDetails `json:"vector"`
	Version string              `json:"version"`
}

type healthCheckResponse struct {
	Details healthCheckDetails `json:"details"`
}

type oldHealthCheckResponse struct {
	Details struct {
		OpenAIEnabled bool `json:"openAI"`
		VectorEnabled bool `json:"vector"`
	} `json:"details"`
}

func (o *openAI) Enabled(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.grafanaURL+appPrefix+"/health", nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.grafanaAPIKey)
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read response body: %w", err)
	}
	var response healthCheckResponse
	if err := json.Unmarshal(body, &response); err != nil {
		// Try the old response format
		var oldResponse oldHealthCheckResponse
		if err := json.Unmarshal(body, &oldResponse); err != nil {
			return false, fmt.Errorf("unmarshal response: %w", err)
		}
		return oldResponse.Details.OpenAIEnabled, nil
	}
	if response.Details.OpenAI.Error != "" {
		err = fmt.Errorf("OpenAI error: %s", response.Details.OpenAI.Error)
	}
	return response.Details.OpenAI.OK, err
}

func (o *openAI) ChatCompletions(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = string(req.Model)
	return o.client.CreateChatCompletion(ctx, r)
}

func (o *openAI) ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	r := req.ChatCompletionRequest
	r.Model = string(req.Model)
	return o.client.CreateChatCompletionStream(ctx, r)
}

func (o *openAI) CreateAssistant(ctx context.Context, req AssistantRequest) (openai.Assistant, error) {
	r := req.AssistantRequest
	r.Model = string(req.Model)
	return o.client.CreateAssistant(ctx, r)
}

func (o *openAI) RetrieveAssistant(ctx context.Context, assistantID string) (openai.Assistant, error) {
	return o.client.RetrieveAssistant(ctx, assistantID)
}

func (o *openAI) ListAssistants(ctx context.Context, limit *int, order *string, after *string, before *string) (openai.AssistantsList, error) {
	return o.client.ListAssistants(ctx, limit, order, after, before)
}

func (o *openAI) DeleteAssistant(ctx context.Context, assistantID string) (openai.AssistantDeleteResponse, error) {
	return o.client.DeleteAssistant(ctx, assistantID)
}

func (o *openAI) CreateThread(ctx context.Context, req openai.ThreadRequest) (openai.Thread, error) {
	return o.client.CreateThread(ctx, req)
}

func (o *openAI) RetrieveThread(ctx context.Context, threadID string) (openai.Thread, error) {
	return o.client.RetrieveThread(ctx, threadID)
}

func (o *openAI) DeleteThread(ctx context.Context, threadID string) (openai.ThreadDeleteResponse, error) {
	return o.client.DeleteThread(ctx, threadID)
}

func (o *openAI) CreateMessage(ctx context.Context, threadID string, request openai.MessageRequest) (msg openai.Message, err error) {
	return o.client.CreateMessage(ctx, threadID, request)
}

func (o *openAI) ListMessages(ctx context.Context, threadID string, limit *int, order *string, after *string, before *string) (msg openai.MessagesList, err error) {
	return o.client.ListMessage(ctx, threadID, limit, order, after, before)
}

func (o *openAI) RetrieveMessage(ctx context.Context, threadID string, messageID string) (msg openai.Message, err error) {
	return o.client.RetrieveMessage(ctx, threadID, messageID)
}

func (o *openAI) DeleteMessage(ctx context.Context, threadID string, messageID string) (msg openai.MessageDeletionStatus, err error) {
	return o.client.DeleteMessage(ctx, threadID, messageID)
}

func (o *openAI) CreateRun(ctx context.Context, threadID string, request openai.RunRequest) (run openai.Run, err error) {
	return o.client.CreateRun(ctx, threadID, request)
}

func (o *openAI) RetrieveRun(ctx context.Context, threadID string, runID string) (run openai.Run, err error) {
	return o.client.RetrieveRun(ctx, threadID, runID)
}

func (o *openAI) CancelRun(ctx context.Context, threadID string, runID string) (run openai.Run, err error) {
	return o.client.CancelRun(ctx, threadID, runID)
}

func (o *openAI) SubmitToolOutputs(ctx context.Context, threadID string, runID string, request openai.SubmitToolOutputsRequest) (response openai.Run, err error) {
	return o.client.SubmitToolOutputs(ctx, threadID, runID, request)
}

func (o *openAI) ListMessage(ctx context.Context, threadID string, limit *int, order *string, after *string, before *string) (openai.MessagesList, error) {
	return o.client.ListMessage(ctx, threadID, limit, order, after, before)
}
