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

	// Try getting the Azure mapping once and fail if we can't.
	_, err := p.getAzureMapping()
	if err != nil {
		return nil, err
	}

	// go-openai expects the URL without the '/openai' suffix, which is
	// the same as us.
	cfg := openai.DefaultAzureConfig(settings.apiKey, settings.URL)
	cfg.HTTPClient = client
	cfg.AzureModelMapperFunc = func(model string) string {
		// We already checked the error when constructing the provider.
		mapping, _ := p.getAzureMapping()
		got := mapping[Model(model)]
		log.DefaultLogger.Debug("mapping model", "from", model, "to", got)
		return got
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

func (p *azure) ChatCompletions(ctx context.Context, req ChatCompletionRequest) (ChatCompletionsResponse, error) {
	mapping, err := p.getAzureMapping()
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	deployment := mapping[req.Model]
	if deployment == "" {
		return ChatCompletionsResponse{}, fmt.Errorf("%w: no deployment found for model: %s", errBadRequest, req.Model)
	}

	u, err := url.Parse(p.settings.URL)
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	u.Path = fmt.Sprintf("/openai/deployments/%s/chat/completions", deployment)
	u.RawQuery = "api-version=2023-03-15-preview"
	reqBody, err := json.Marshal(azureChatCompletionRequest{
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
	httpReq.Header.Set("api-key", p.settings.apiKey)
	return doOpenAIRequest(p.c, httpReq)
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

func (p *azure) StreamChatCompletions(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	r := req.ChatCompletionRequest
	r.Model = string(req.Model)
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
