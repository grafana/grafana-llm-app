package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sashabaranov/go-openai"
)

type azure struct {
	settings OpenAISettings
	c        *http.Client
	oc       *openai.Client
}

func NewAzureProvider(settings OpenAISettings) (LLMProvider, error) {
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}
	p := &azure{
		settings: settings,
		c:        client,
	}

	// go-openai expects the URL without the '/openai' suffix, which is
	// the same as us.
	cfg := openai.DefaultAzureConfig(settings.apiKey, settings.URL)
	cfg.HTTPClient = client
	// We pass the deployment as the name of the model, so just return the untransformed string.
	cfg.AzureModelMapperFunc = func(model string) string {
		return model
	}

	p.oc = openai.NewClientWithConfig(cfg)
	return p, nil
}

func (p *azure) Models(ctx context.Context) (ModelResponse, error) {
	models := make([]ModelInfo, 0, len(p.settings.AzureMapping))
	mapping, err := p.getAzureMapping()
	if err != nil {
		return ModelResponse{}, err
	}
	for model := range mapping {
		models = append(models, ModelInfo{ID: model})
	}
	return ModelResponse{Data: models}, nil
}

type azureChatCompletionRequest struct {
	ChatCompletionRequest
	// Azure does not use the model field.
	Model string `json:"-"`
}

func (p *azure) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	mapping, err := p.getAzureMapping()
	if err != nil {
		return ChatCompletionResponse{}, err
	}
	deployment := mapping[req.Model]
	if deployment == "" {
		return ChatCompletionResponse{}, fmt.Errorf("%w: no deployment found for model: %s", errBadRequest, req.Model)
	}

	u, err := url.Parse(p.settings.URL)
	if err != nil {
		return ChatCompletionResponse{}, err
	}
	u.Path = fmt.Sprintf("/openai/deployments/%s/chat/completions", deployment)
	u.RawQuery = "api-version=2023-03-15-preview"
	reqBody, err := json.Marshal(azureChatCompletionRequest{
		ChatCompletionRequest: req,
		Model:                 req.Model.toOpenAI(),
	})
	if err != nil {
		return ChatCompletionResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return ChatCompletionResponse{}, err
	}
	httpReq.Header.Set("api-key", p.settings.apiKey)
	return doOpenAIRequest(p.c, httpReq)
}

func (p *azure) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	mapping, err := p.getAzureMapping()
	if err != nil {
		return nil, err
	}
	deployment := mapping[req.Model]
	if deployment == "" {
		return nil, fmt.Errorf("%w: no deployment found for model: %s", errBadRequest, req.Model)
	}

	r := req.ChatCompletionRequest
	// For the Azure mapping we want to use the name of the mapped deployment as the model.
	r.Model = deployment
	return streamOpenAIRequest(ctx, r, p.oc)
}

func (p *azure) getAzureMapping() (map[Model]string, error) {
	result := make(map[Model]string, len(p.settings.AzureMapping))
	for _, v := range p.settings.AzureMapping {
		if len(v) != 2 {
			return nil, fmt.Errorf("%w: expected 2 entries in a mapping, got %d", errBadRequest, len(v))
		}
		model, err := ModelFromString(v[0])
		if err != nil {
			return nil, err
		}
		result[model] = v[1]
	}
	return result, nil
}
