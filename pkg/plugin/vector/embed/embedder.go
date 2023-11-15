package embed

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type EmbedderType string

const (
	EmbedderOpenAI           EmbedderType = "openai"
	EmbedderGrafanaVectorAPI EmbedderType = "grafana/vectorapi"
)

type Embedder interface {
	Embed(ctx context.Context, model string, text string) ([]float32, error)
	Health(ctx context.Context, model string) error
}

type Settings struct {
	Type EmbedderType `json:"type"`

	OpenAI                   openAISettings
	GrafanaVectorAPISettings grafanaVectorAPISettings `json:"grafanaVectorAPI"`
}

// NewEmbedder creates a new embedder.
func NewEmbedder(s Settings, secrets map[string]string) (Embedder, error) {
	log.DefaultLogger.Debug("Creating OpenAI embedder")
	// Grafana Vector API embedder is OpenAI compatible so we can reuse the client
	// The EmbedderType is used in settings.load_settings to duplicate the correct OpenAI settings
	return newOpenAIEmbedder(s, secrets), nil
}
