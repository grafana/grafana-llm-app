package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type openAI struct {
	settings OpenAISettings
	c        *http.Client
}

func NewOpenAIProvider(settings OpenAISettings) LLMProvider {
	return &openAI{
		settings: settings,
		c: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (p *openAI) Models(ctx context.Context) (ModelResponse, error) {
	return ModelResponse{
		Data: []ModelInfo{
			{ID: ModelSmall},
			{ID: ModelMedium},
			{ID: ModelLarge},
		},
	}, nil
}

type openAIChatCompletionRequest struct {
	ChatCompletionRequest
	// Override the model field to just be a string rather than our custom Model type.
	Model string `json:"model"`
}

func (p *openAI) ChatCompletions(ctx context.Context, req ChatCompletionRequest) (ChatCompletionsResponse, error) {
	u, err := url.Parse(p.settings.URL)
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	u.Path, err = url.JoinPath(u.Path, "v1/chat/completions")
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
	httpReq.Header.Set("Authorization", "Bearer "+p.settings.apiKey)
	httpReq.Header.Set("OpenAI-Organization", p.settings.OrganizationID)
	return doOpenAIRequest(p.c, httpReq)
}

func doOpenAIRequest(c *http.Client, req *http.Request) (ChatCompletionsResponse, error) {
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatCompletionsResponse{}, err
	}

	if resp.StatusCode/100 != 2 {
		return ChatCompletionsResponse{}, fmt.Errorf("error from OpenAI: %d, %s", resp.StatusCode, string(respBody))
	}

	completions := ChatCompletionsResponse{}
	err = json.Unmarshal(respBody, &completions)
	if err != nil {
		return ChatCompletionsResponse{}, err
	}
	return completions, nil
}
