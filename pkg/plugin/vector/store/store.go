package store

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type VectorStoreType string

const (
	VectorStoreTypeQdrant           VectorStoreType = "qdrant"
	VectorStoreTypeGrafanaVectorAPI VectorStoreType = "grafana/vectorapi"
)

type Collection struct {
	Name string `json:"name"`
}

type Payload map[string]any

type SearchResult struct {
	Payload Payload `json:"payload"`
	Score   float64 `json:"score"`
}

type ReadVectorStore interface {
	CollectionExists(ctx context.Context, collection string) (bool, error)
	Search(ctx context.Context, collection string, vector []float32, topK uint64, filter map[string]interface{}) ([]SearchResult, error)
	Health(ctx context.Context) error
}

type WriteVectorStore interface {
	CreateCollection(ctx context.Context, collection string, size uint64) error
	PointExists(ctx context.Context, collection string, id uint64) (bool, error)
	UpsertColumnar(ctx context.Context, collection string, ids []uint64, embeddings [][]float32, payloads []Payload) error
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

func NewVectorStore(s Settings, secrets map[string]string) (VectorStore, context.CancelFunc, error) {
	switch VectorStoreType(s.Type) {
	case VectorStoreTypeGrafanaVectorAPI:
		log.DefaultLogger.Debug("Grafana Vector API can not yet be used for vector sync")
		return nil, nil, fmt.Errorf("unimplemented")
	case VectorStoreTypeQdrant:
		log.DefaultLogger.Debug("Creating Qdrant store")
		return newQdrantStore(s.Qdrant, secrets)
	}
	return nil, nil, nil
}
