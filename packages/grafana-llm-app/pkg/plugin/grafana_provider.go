package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

type grafanaProvider struct {
	settings LLMGatewaySettings
	tenant   string
	gcomKey  string
	c        *http.Client
	oc       *openai.Client
}

type TenantRoundTripper struct {
	next   http.RoundTripper
	tenant string
}

func (t *TenantRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-Scope-OrgID", t.tenant)
	return t.next.RoundTrip(r)
}

func NewGrafanaProvider(settings Settings) (LLMProvider, error) {
	client := &http.Client{
		Timeout:   2 * time.Minute,
		Transport: &TenantRoundTripper{next: http.DefaultTransport, tenant: settings.Tenant},
	}
	cfg := openai.DefaultConfig(fmt.Sprintf("%s:%s", settings.Tenant, settings.GrafanaComAPIKey))
	base, err := url.JoinPath(settings.LLMGateway.URL, "/openai/v1")
	if err != nil {
		return nil, fmt.Errorf("join url: %w", err)
	}
	cfg.BaseURL = base
	cfg.HTTPClient = client
	return &grafanaProvider{
		settings: settings.LLMGateway,
		tenant:   settings.Tenant,
		gcomKey:  settings.GrafanaComAPIKey,
		c:        client,
		oc:       openai.NewClientWithConfig(cfg),
	}, nil
}

func (p *grafanaProvider) Models(ctx context.Context) (ModelResponse, error) {
	return ModelResponse{
		Data: []ModelInfo{
			{ID: ModelSmall},
			{ID: ModelMedium},
			{ID: ModelLarge},
		},
	}, nil
}

func (p *grafanaProvider) ChatCompletions(ctx context.Context, req ChatCompletionRequest) (ChatCompletionsResponse, error) {
	u, err := url.Parse(p.settings.URL)
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	// We keep the openai prefix when using llm-gateway.
	u.Path, err = url.JoinPath(u.Path, "openai/v1/chat/completions")
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	reqBody, err := json.Marshal(openAIChatCompletionRequest{
		ChatCompletionRequest: req,
		Model:                 req.Model.toOpenAI(),
	})
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	httpReq.SetBasicAuth(p.tenant, p.gcomKey)
	return doOpenAIRequest(p.c, httpReq)
}

func (p *grafanaProvider) StreamChatCompletions(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toOpenAI()
	stream, err := p.oc.CreateChatCompletionStream(ctx, r)
	if err != nil {
		log.DefaultLogger.Error("error establishing stream", "err", err)
		return nil, err
	}
	c := make(chan ChatCompletionStreamResponse)

	go func() {
		defer stream.Close()
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
