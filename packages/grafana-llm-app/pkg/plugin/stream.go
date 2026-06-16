package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/grafana-llm-app/pkg/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const (
	openAIChatCompletionsPath = "openai/v1/chat/completions" // Deprecated
	llmChatCompletionsPath    = "llm/v1/chat/completions"
	mcpPath                   = "mcp"
)

// allowMCPRequest returns true if the request path is for the MCP server
// and the MCP server is enabled and running.
func (a *App) allowMCPRequest(path string) bool {
	return strings.HasPrefix(path, mcpPath) && !a.settings.MCP.Disabled && a.mcpServer != nil && a.mcpServer.LiveServer != nil
}

func (a *App) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	log.DefaultLogger.Debug(fmt.Sprintf("SubscribeStream: %s", req.Path))

	resp := &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusNotFound,
	}

	// Backwards compatibility for old paths
	if strings.HasPrefix(req.Path, llmChatCompletionsPath) || strings.HasPrefix(req.Path, openAIChatCompletionsPath) {
		resp.Status = backend.SubscribeStreamStatusOK
	}
	if a.allowMCPRequest(req.Path) {
		resp.Status = backend.SubscribeStreamStatusOK
	}
	return resp, nil
}

func (a *App) runChatCompletionsStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	requestBody := ChatCompletionRequest{}
	var err error
	err = json.Unmarshal(req.Data, &requestBody)
	if err != nil {
		return fmt.Errorf("unable to unmarshal request body: %w", err)
	}

	llmProvider, err := createProvider(a.settings)
	if err != nil {
		return err
	}

	// Always set stream to true for streaming requests.
	requestBody.Stream = true

	// Delegate to configured provider for chat completions stream.
	c, err := llmProvider.ChatCompletionStream(ctx, requestBody)
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
	err = sender.SendJSON([]byte(`{"choices": [{"delta": {"done": true}}]}`))
	if err != nil {
		return fmt.Errorf("send stream data: %w", err)
	}
	return nil
}

func (a *App) runMCPStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	return a.mcpServer.LiveServer.HandleStream(ctx, req, sender)
}

func (a *App) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Debug(fmt.Sprintf("RunStream: %s", req.Path), "data", string(req.Data))

	// Backwards compatibility for old paths
	if strings.HasPrefix(req.Path, openAIChatCompletionsPath) || strings.HasPrefix(req.Path, llmChatCompletionsPath) {
		// Run the stream. On error, send an error message over the stream sender, then return.
		// We want to avoid returning an `error` here as much as possible because Grafana will
		// blindly rerun the stream without notifying the UI if we do.
		if err := a.runChatCompletionsStream(ctx, req, sender); err != nil {
			log.DefaultLogger.Error("error running stream", "provider", a.settings.Provider, "err", err)
			sendError(EventError{Error: err.Error()}, sender)
		}
		return nil
	}
	if a.allowMCPRequest(req.Path) {
		if err := a.runMCPStream(ctx, req, sender); err != nil {
			log.DefaultLogger.Error("error running stream", "err", err)
			sendError(EventError{Error: err.Error()}, sender)
		}
		return nil
	}
	return fmt.Errorf("unknown stream path: %s", req.Path)
}

// PublishStream handles messages sent to the PublishStream handler.
func (a *App) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	log.DefaultLogger.Debug(fmt.Sprintf("PublishStream: %s", req.Path), "data", string(req.Data))
	// Handle messages for the MCP server.
	if a.allowMCPRequest(req.Path) {
		err := a.mcpServer.LiveServer.HandleMessage(ctx, req)
		if errors.Is(err, mcp.ErrStreamNotFound) {
			log.DefaultLogger.Error("MCP stream not found", "err", err, "path", req.Path)
			return &backend.PublishStreamResponse{
				Status: backend.PublishStreamStatusNotFound,
			}, nil
		}
		if err != nil {
			return nil, err
		}
		return &backend.PublishStreamResponse{Status: backend.PublishStreamStatusOK}, nil
	}
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
