package embed

import "context"

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
func NewEmbedder(s Settings) (Embedder, error) {
	switch EmbedderType(s.Type) {
	case EmbedderOpenAI:
		return newOpenAIEmbedder(s.OpenAI), nil
	}
	return nil, nil
}
