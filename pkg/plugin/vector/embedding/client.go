package embedding

import "context"

type EmbeddingClient interface {
	Embeddings(ctx context.Context, text string) ([]float32, error)
}

type EmbeddingClientSettings struct {
	Type              string            `json:"type"`
	YasEngineSettings YasEngineSettings `json:"yas"`
}

type YasEngineSettings struct {
	URL string `json:"url"`
}

func NewEmbeddingClient(s EmbeddingClientSettings) (EmbeddingClient, error) {
	return nil, nil
}
