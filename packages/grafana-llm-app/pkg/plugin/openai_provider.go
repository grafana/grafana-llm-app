package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

// customAuthTransport injects the configured auth header on every request.
// go-openai sets "Authorization: Bearer <token>" inside sendRequest before calling HTTPClient.Do,
// so we pass DefaultConfig("") (empty token) and let this transport override the header.
type customAuthTransport struct {
	base       http.RoundTripper
	headerName string
	apiKey     string
}

func (t *customAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r2 := req.Clone(req.Context())
	r2.Header.Del("Authorization") // remove the empty "Bearer " set by DefaultConfig("")
	if strings.EqualFold(t.headerName, "Authorization") {
		r2.Header.Set("Authorization", "Bearer "+t.apiKey)
	} else {
		r2.Header.Set(t.headerName, t.apiKey)
	}
	return t.base.RoundTrip(r2)
}

type openAI struct {
	settings OpenAISettings
	models   *ModelSettings
	oc       *openai.Client
}

func NewOpenAIProvider(settings OpenAISettings, models *ModelSettings) (LLMProvider, error) {
	// Defensively check that APIPath is not nil to avoid potential panics
	// if settings aren't loaded using loadSettings.
	if settings.APIPath == nil {
		settings.APIPath = &defaultOpenAIAPIPath
	}
	if settings.AuthHeaderName == "" {
		settings.AuthHeaderName = "Authorization"
	}

	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
		Transport: &customAuthTransport{
			base:       http.DefaultTransport,
			headerName: settings.AuthHeaderName,
			apiKey:     settings.apiKey,
		},
	}

	// Pass empty auth token so go-openai doesn't set its own Authorization header;
	// customAuthTransport handles auth injection for all header name variants.
	cfg := openai.DefaultConfig("")
	base, err := url.JoinPath(settings.URL, *settings.APIPath)
	if err != nil {
		return nil, fmt.Errorf("join url: %w", err)
	}
	cfg.BaseURL = base
	cfg.HTTPClient = httpClient
	cfg.OrgID = settings.OrganizationID
	return &openAI{
		settings: settings,
		models:   models,
		oc:       openai.NewClientWithConfig(cfg),
	}, nil
}

func (p *openAI) Models(ctx context.Context) (ModelResponse, error) {
	return ModelResponse{
		Data: []ModelInfo{
			{ID: ModelBase},
			{ID: ModelLarge},
		},
	}, nil
}

func (p *openAI) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toOpenAI(p.models)

	ForceUserMessage(&r)

	resp, err := p.oc.CreateChatCompletion(ctx, r)
	if err != nil {
		log.DefaultLogger.Error("error creating openai chat completion", "err", err)
		return openai.ChatCompletionResponse{}, err
	}
	return resp, nil
}

func (p *openAI) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toOpenAI(p.models)

	ForceUserMessage(&r)

	return streamOpenAIRequest(ctx, r, p.oc)
}

func streamOpenAIRequest(ctx context.Context, r openai.ChatCompletionRequest, oc *openai.Client) (<-chan ChatCompletionStreamResponse, error) {
	r.Stream = true

	ForceUserMessage(&r)

	stream, err := oc.CreateChatCompletionStream(ctx, r)
	if err != nil {
		log.DefaultLogger.Error("error establishing stream", "err", err)
		return nil, err
	}
	c := make(chan ChatCompletionStreamResponse)

	go func() {
		defer stream.Close() //nolint:errcheck
		defer close(c)
		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				log.DefaultLogger.Error("openai stream error", "err", err)
				c <- ChatCompletionStreamResponse{Error: err}
				return
			}

			c <- ChatCompletionStreamResponse{ChatCompletionStreamResponse: resp}
		}
	}()
	return c, nil
}
