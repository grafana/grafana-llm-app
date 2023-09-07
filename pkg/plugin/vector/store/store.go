package store

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type VectorStoreType string

const (
	VectorStoreTypeGrafanaVectorAPI VectorStoreType = "grafana/vectorapi"
)

type Collection struct {
	Name      string `json:"name"`
	Dimension int    `json:"dimension"`
	Model     string `json:"model"`
}

type SearchResult struct {
	Payload map[string]any `json:"payload"`
	Score   float64        `json:"score"`
}

type ReadVectorStore interface {
	Collections(ctx context.Context) ([]string, error)
	Search(ctx context.Context, collection string, vector []float32, limit uint64) ([]SearchResult, error)
}

type WriteVectorStore interface {
	Collections(ctx context.Context) ([]string, error)
	CollectionExists(ctx context.Context, collection string) (bool, error)
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

	Collections []Collection `json:"collections"`
}

func NewReadVectorStore(s Settings) (ReadVectorStore, error) {
	switch VectorStoreType(s.Type) {
	case VectorStoreTypeGrafanaVectorAPI:
		log.DefaultLogger.Debug("Creating Grafana Vector API store")
		return newGrafanaVectorAPI(s.GrafanaVectorAPI), nil
	}
	return nil, nil
}

func NewVectorStore(s Settings) (VectorStore, error) {
	// TODO: Implement write vector store.
	return nil, nil
}
