package plugin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

type azure struct {
	settings OpenAISettings
	oc       *openai.Client
}

func NewAzureProvider(settings OpenAISettings) (LLMProvider, error) {
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}
	p := &azure{
		settings: settings,
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

func (p *azure) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	mapping, err := p.getAzureMapping()
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}
	deployment := mapping[req.Model]
	if deployment == "" {
		return openai.ChatCompletionResponse{}, fmt.Errorf("%w: no deployment found for model: %s", errBadRequest, req.Model)
	}

	r := req.ChatCompletionRequest
	r.Model = deployment
	resp, err := p.oc.CreateChatCompletion(ctx, r)
	if err != nil {
		log.DefaultLogger.Error("error creating azure chat completion", "err", err)
		return openai.ChatCompletionResponse{}, err
	}
	return resp, nil
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
