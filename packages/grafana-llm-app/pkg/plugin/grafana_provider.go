package plugin

import (
	"context"
	"fmt"
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
		oc:       openai.NewClientWithConfig(cfg),
	}, nil
}

func (p *grafanaProvider) Models(ctx context.Context) (ModelResponse, error) {
	return ModelResponse{
		Data: []ModelInfo{
			{ID: ModelBase},
			{ID: ModelLarge},
		},
	}, nil
}

func (p *grafanaProvider) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toOpenAI(defaultModelSettings(ProviderTypeGrafana))

	ForceUserMessage(&r)

	resp, err := p.oc.CreateChatCompletion(ctx, r)
	if err != nil {
		log.DefaultLogger.Error("error creating grafana chat completion", "err", err)
		return openai.ChatCompletionResponse{}, err
	}
	return resp, nil
}

func (p *grafanaProvider) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = req.Model.toOpenAI(defaultModelSettings(ProviderTypeGrafana))

	ForceUserMessage(&r)

	return streamOpenAIRequest(ctx, r, p.oc)
}
