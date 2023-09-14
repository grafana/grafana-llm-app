package store

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type VectorStoreType string

const (
	VectorStoreTypeQdrant           VectorStoreType = "qdrant"
	VectorStoreTypeGrafanaVectorAPI VectorStoreType = "grafana/vectorapi"
)

type SearchResult struct {
	Payload map[string]any `json:"payload"`
	Score   float64        `json:"score"`
}

type ReadVectorStore interface {
	CollectionExists(ctx context.Context, collection string) (bool, error)
	Search(ctx context.Context, collection string, vector []float32, topK uint64) ([]SearchResult, error)
}

type WriteVectorStore interface {
	Collections(ctx context.Context) ([]string, error)
	CreateCollection(ctx context.Context, collection string, size uint64) error
	PointExists(ctx context.Context, collection string, id uint64) (bool, error)
	UpsertColumnar(ctx context.Context, collection string, ids []uint64, embeddings [][]float32, payloadJSONs []string) error
}

type VectorStore interface {
	ReadVectorStore
	WriteVectorStore
}

type Settings struct {
	Type string `json:"type"`

	GrafanaVectorAPI grafanaVectorAPISettings `json:"grafanaVectorAPI"`

	Qdrant qdrantSettings `json:"qdrant"`
}

func NewReadVectorStore(s Settings, secrets map[string]string) (ReadVectorStore, context.CancelFunc, error) {
	switch VectorStoreType(s.Type) {
	case VectorStoreTypeGrafanaVectorAPI:
		log.DefaultLogger.Debug("Creating Grafana Vector API store")
		return newGrafanaVectorAPI(s.GrafanaVectorAPI, secrets), func() {}, nil
	case VectorStoreTypeQdrant:
		log.DefaultLogger.Debug("Creating Qdrant store")
		return newQdrantStore(s.Qdrant, secrets)
	}
	return nil, nil, nil
}

func NewVectorStore(s Settings) (VectorStore, error) {
	// TODO: Implement write vector store.
	return nil, nil
}
