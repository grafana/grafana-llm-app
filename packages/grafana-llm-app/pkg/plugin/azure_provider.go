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
	// We need the model settings to determine the default model in case it's
	// missing from a request.
	modelSettings ModelSettings
	oc            *openai.Client
}

func NewAzureProvider(settings OpenAISettings, modelSettings *ModelSettings) (LLMProvider, error) {
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}
	p := &azure{
		settings:      settings,
		modelSettings: *modelSettings,
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

func (p *azure) getDeployment(model Model) (string, error) {
	mapping, err := p.getAzureMapping()
	if err != nil {
		return "", err
	}
	if model == "" {
		model = p.modelSettings.Default
	}
	deployment := mapping[model]
	if deployment == "" {
		return "", fmt.Errorf("%w: no deployment found for model: %s", errBadRequest, model)
	}
	return deployment, nil
}

func (p *azure) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	deployment, err := p.getDeployment(req.Model)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
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
	deployment, err := p.getDeployment(req.Model)
	if err != nil {
		return nil, err
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
