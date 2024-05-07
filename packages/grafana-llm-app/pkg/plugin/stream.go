package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const openAIChatCompletionsPath = "openai/v1/chat/completions"

func (a *App) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	log.DefaultLogger.Debug(fmt.Sprintf("SubscribeStream: %s", req.Path))

	resp := &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusNotFound,
	}
	if strings.HasPrefix(req.Path, openAIChatCompletionsPath) {
		resp.Status = backend.SubscribeStreamStatusOK
	}
	return resp, nil
}

func (a *App) runOpenAIChatCompletionsStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	requestBody := ChatCompletionRequest{}
	var err error
	err = json.Unmarshal(req.Data, &requestBody)
	if err != nil {
		return fmt.Errorf("Unable to unmarshal request body: %w", err)
	}

	// Always set stream to true for streaming requests.
	requestBody.Stream = true

	// Delegate to configured provider for chat completions stream.
	c, err := a.llmProvider.StreamChatCompletions(ctx, requestBody)
	if err != nil {
		return fmt.Errorf("establish chat completions stream: %w", err)
	}
	// Send all messages to the sender.
	for resp := range c {
		if resp.Error != nil {
			return resp.Error
		}
		data, err := json.Marshal(resp)
		if err != nil {
			return fmt.Errorf("marshal chat completions stream response: %w", err)
		}
		err = sender.SendJSON(data)
		if err != nil {
			return fmt.Errorf("send stream data: %w", err)
		}
	}
	// Finish with a done message for compatibility.
	// Clients will use this to know when to unsubscribe to the stream.
	err = sender.SendJSON([]byte(`{"done": true}`))
	if err != nil {
		return fmt.Errorf("send stream data: %w", err)
	}
	return nil
}

func (a *App) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Debug(fmt.Sprintf("RunStream: %s", req.Path), "data", string(req.Data))
	if strings.HasPrefix(req.Path, openAIChatCompletionsPath) {
		// Run the stream. On error, send an error message over the stream sender, then return.
		// We want to avoid returning an `error` here as much as possible because Grafana will
		// blindly rerun the stream without notifying the UI if we do.
		if err := a.runOpenAIChatCompletionsStream(ctx, req, sender); err != nil {
			log.DefaultLogger.Error("error running stream", "err", err)
			sendError(EventError{Error: err.Error()}, sender)
		}
		return nil
	}
	return fmt.Errorf("unknown stream path: %s", req.Path)
}

func (a *App) PublishStream(context.Context, *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}

type EventDone struct {
	Done bool `json:"done"`
}

type EventError struct {
	Error string `json:"error"`
}

func sendError(event EventError, sender *backend.StreamSender) {
	err := fmt.Errorf("proxy: stream: error from event stream: %s", event.Error)
	log.DefaultLogger.Error(err.Error())
	b, err := json.Marshal(event)
	if err != nil {
		err = fmt.Errorf("proxy: stream: error marshalling error payload: %w", err)
		log.DefaultLogger.Error(err.Error())
		b = []byte(fmt.Sprintf(`{"error": "%s", "done": false}`, err.Error()))
	}
	if err = sender.SendJSON(b); err != nil {
		err = fmt.Errorf("proxy: stream: error unmarshalling event data: %w", err)
		log.DefaultLogger.Error(err.Error())
	}
}
