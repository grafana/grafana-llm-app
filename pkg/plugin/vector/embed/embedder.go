package embed

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type EmbedderType string

const (
	EmbedderOpenAI EmbedderType = "openai"
)

type Embedder interface {
	Embed(ctx context.Context, model string, text string) ([]float32, error)
}

type Settings struct {
	Type string `json:"type"`

	OpenAI openAISettings `json:"openai"`
}

// NewEmbedder creates a new embedder.
func NewEmbedder(s Settings, secrets map[string]string) (Embedder, error) {
	switch EmbedderType(s.Type) {
	case EmbedderOpenAI:
		log.DefaultLogger.Debug("Creating OpenAI embedder")
		return newOpenAIEmbedder(s.OpenAI, secrets), nil
	}
	return nil, nil
}
