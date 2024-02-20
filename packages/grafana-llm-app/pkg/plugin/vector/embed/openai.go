package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type openAISettings struct {
	URL      string
	AuthType string
}

type grafanaVectorAPISettings struct {
	URL           string `json:"url"`
	AuthType      string `json:"authType"`
	BasicAuthUser string `json:"basicAuthUser"`
}

type openAIEmbeddingsAuthSettings struct {
	BasicAuthUser     string
	BasicAuthPassword string
	OpenAIKey         string
}

type openAIClient struct {
	client       *http.Client
	url          string
	authType     string
	providerType EmbedderType
	authSettings openAIEmbeddingsAuthSettings
}

type openAIEmbeddingsRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIEmbeddingsResponse struct {
	Data []openAIEmbeddingData `json:"data"`
}

type openAIEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
}

func (o *openAIClient) setAuth(req *http.Request) {
	switch o.authType {
	case "basic-auth":
		req.SetBasicAuth(o.authSettings.BasicAuthUser, o.authSettings.BasicAuthPassword)
	case "openai-key-auth":
		req.Header.Add("Authorization", "Bearer "+o.authSettings.OpenAIKey)
	}

}

func (o *openAIClient) getProviderString() string {
	switch o.providerType {
	case EmbedderOpenAI:
		return "OpenAI"
	case EmbedderGrafanaVectorAPI:
		return "Grafana Vector API"
	default:
		return "Unknown"
	}
}

func (o *openAIClient) Embed(ctx context.Context, model string, payload string) ([]float32, error) {
	// TODO: ensure payload is under 8191 tokens, somehow.
	url := o.url
	if url == "" {
		url = "https://api.openai.com"
	}
	url = strings.TrimSuffix(url, "/")
	url = url + "/v1/embeddings"
	reqBody := openAIEmbeddingsRequest{
		Model: model,
		Input: payload,
	}
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	o.setAuth(req)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.DefaultLogger.Warn("failed to close response body", "err", err)
		}
	}()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("got non-2xx status from %s: %s", o.getProviderString(), resp.Status)
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024*2))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	var body openAIEmbeddingsResponse
	err = json.Unmarshal(respBody, &body)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response body: %w", err)
	}
	if len(body.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return body.Data[0].Embedding, nil
}

func (o *openAIClient) Health(ctx context.Context, model string) error {
	_, err := o.Embed(ctx, model, "Hello, world!")
	return err
}

// newOpenAIEmbedder creates a new Embedder using OpenAI's API.
func newOpenAIEmbedder(settings Settings, secrets map[string]string) Embedder {
	var impl openAIClient
	switch settings.Type {
	case EmbedderOpenAI:
		impl = openAIClient{
			client:       &http.Client{},
			url:          settings.OpenAI.URL,
			authType:     string(settings.OpenAI.AuthType),
			providerType: settings.Type,
			authSettings: openAIEmbeddingsAuthSettings{
				OpenAIKey: secrets["openAIKey"],
			},
		}
	case EmbedderGrafanaVectorAPI:
		impl = openAIClient{
			client:       &http.Client{},
			url:          settings.GrafanaVectorAPISettings.URL,
			authType:     string(settings.GrafanaVectorAPISettings.AuthType),
			providerType: settings.Type,
			authSettings: openAIEmbeddingsAuthSettings{
				BasicAuthUser:     settings.GrafanaVectorAPISettings.BasicAuthUser,
				BasicAuthPassword: secrets["vectorEmbedderBasicAuthPassword"],
			},
		}
	default:
		return nil
	}

	return &impl
}
