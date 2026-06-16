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

type VectorStoreAuthType string

const (
	VectorStoreAuthTypeBasicAuth VectorStoreAuthType = "basic-auth"
)

type SearchResult struct {
	Payload map[string]any `json:"payload"`
	Score   float64        `json:"score"`
}

type ReadVectorStore interface {
	CollectionExists(ctx context.Context, collection string) (bool, error)
	Search(ctx context.Context, collection string, vector []float32, topK uint64, filter map[string]interface{}) ([]SearchResult, error)
	Health(ctx context.Context) error
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

type AuthSettings struct {
	BasicAuthUser string `json:"basicAuthUser"`
}

type Settings struct {
	Type VectorStoreType `json:"type"`

	GrafanaVectorAPI GrafanaVectorAPISettings `json:"grafanaVectorAPI"`

	Qdrant qdrantSettings `json:"qdrant"`
}

func NewReadVectorStore(s Settings, secrets map[string]string) (ReadVectorStore, context.CancelFunc, error) {
	switch s.Type {
	case VectorStoreTypeGrafanaVectorAPI:
		log.DefaultLogger.Debug("Creating Grafana Vector API store")
		vectorStore, err := newGrafanaVectorAPI(s.GrafanaVectorAPI, secrets)
		return vectorStore, func() {}, err
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
