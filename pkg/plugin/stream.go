package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/launchdarkly/eventsource"
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

	settings := loadSettings(*req.PluginContext.AppInstanceSettings)
	requestBody := map[string]interface{}{}
	var err error
	err = json.Unmarshal(req.Data, &requestBody)

	if err != nil {
		return fmt.Errorf("Unable to unmarshal request body: %w", err)
	}

	// set stream to true
	requestBody["stream"] = true

	u, err := url.Parse(settings.OpenAI.URL)

	if err != nil {
		return fmt.Errorf("Unable to parse OpenAI URL: %w", err)
	}

	var outgoingBody []byte

	if settings.OpenAI.UseAzure {
		// Map model to deployment

		var deployment string = ""
		for _, v := range settings.OpenAI.AzureMapping {
			if val, ok := requestBody["model"].(string); ok && val == v[0] {
				deployment = v[1]
				break
			}
		}

		if deployment == "" {
			return fmt.Errorf("No deployment found for model: %s", requestBody["model"])
		}

		u.Path = fmt.Sprintf("/openai/deployments/%s/chat/completion", deployment)
		u.RawQuery = "api-version=2023-03-15-preview"

		// Remove extra fields
		delete(requestBody, "model")

	} else {
		u.Path = strings.TrimPrefix(req.Path, "openai")
	}

	outgoingBody, err = json.Marshal(requestBody)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(outgoingBody))
	if err != nil {
		return fmt.Errorf("proxy: stream: error creating request: %w", err)
	}

	if settings.OpenAI.UseAzure {
		httpReq.Header.Set("api-key", settings.OpenAI.apiKey)
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+settings.OpenAI.apiKey)
		httpReq.Header.Set("OpenAI-Organization", settings.OpenAI.OrganizationID)
	}

	lastEventID := "" // no last event id
	eventStream, err := eventsource.SubscribeWithRequest(lastEventID, httpReq)
	if err != nil {
		return fmt.Errorf("proxy: stream: eventsource.SubscribeWithRequest: %s: %w", httpReq.URL, err)
	}
	defer func() {
		eventStream.Close()
		log.DefaultLogger.Debug(fmt.Sprintf("proxy: stream: stream closed: %s", req.Path))
	}()

	// Stream response back to frontend.
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-eventStream.Events:
			var body map[string]interface{}
			eventData := event.Data()
			// If the event data is "[DONE]", then we're done.
			if eventData == "[DONE]" {
				err = sender.SendJSON([]byte(`{"choices": [{"delta": {"done": true}}]}`))
				if err != nil {
					err = fmt.Errorf("proxy: stream: error sending done: %w", err)
					log.DefaultLogger.Error(err.Error())
					return err
				}
				log.DefaultLogger.Debug(fmt.Sprintf("proxy: stream: done==true, ending (in happy branch): %s", req.Path))
				return nil
			}
			// Make sure we can unmarshal the data.
			err = json.Unmarshal([]byte(eventData), &body)
			if err != nil {
				err = fmt.Errorf("proxy: stream: error unmarshalling event data %s: %w", eventData, err)
				log.DefaultLogger.Error(err.Error())
				return err
			}
			err = sender.SendJSON([]byte(event.Data()))
			if err != nil {
				err = fmt.Errorf("proxy: stream: error sending event data: %w", err)
				log.DefaultLogger.Error(err.Error())
				return err
			}
		case err := <-eventStream.Errors:
			err = fmt.Errorf("proxy: stream: error from event stream: %w", err)
			log.DefaultLogger.Error(err.Error())
			var payload struct {
				Error string `json:"error"`
				Done  bool   `json:"done"`
			}
			payload.Error = err.Error()
			b, err := json.Marshal(payload)
			if err != nil {
				err = fmt.Errorf("proxy: stream: error marshalling error payload: %w", err)
				log.DefaultLogger.Error(err.Error())
				return err
			}
			err = sender.SendJSON([]byte(b))
			if err != nil {
				err = fmt.Errorf("proxy: stream: error unmarshalling event data: %w", err)
				log.DefaultLogger.Error(err.Error())
				return err
			}
			if payload.Done { // graceful end
				log.DefaultLogger.Debug(fmt.Sprintf("proxy: stream: done==true, ending (in error branch): %s", req.Path))
				return nil
			}
			return err
		}
	}
}

func (a *App) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	log.DefaultLogger.Debug(fmt.Sprintf("RunStream: %s", req.Path), "data", string(req.Data))
	if strings.HasPrefix(req.Path, openAIChatCompletionsPath) {
		return a.runOpenAIChatCompletionsStream(ctx, req, sender)
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
