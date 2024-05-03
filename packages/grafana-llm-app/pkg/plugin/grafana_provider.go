package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type grafanaProvider struct {
	settings LLMGatewaySettings
	tenant   string
	gcomKey  string
	c        *http.Client
}

func NewGrafanaProvider(settings Settings) LLMProvider {
	return &grafanaProvider{
		settings: settings.LLMGateway,
		tenant:   settings.Tenant,
		gcomKey:  settings.GrafanaComAPIKey,
		c: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (p *grafanaProvider) Models(ctx context.Context) (ModelResponse, error) {
	return ModelResponse{
		Data: []ModelInfo{
			{ID: ModelDefault},
			{ID: ModelHighAccuracy},
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
	httpReq.Header.Add("X-Scope-OrgID", p.tenant)
	return doOpenAIRequest(p.c, httpReq)
}
