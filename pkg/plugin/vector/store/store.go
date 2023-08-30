package store

import (
	"context"
)

type VectorStoreClient interface {
	Collections(ctx context.Context) ([]string, error)
	CollectionExists(ctx context.Context, collection string) (bool, error)
	CreateCollection(ctx context.Context, collection string, size uint64) error
	PointExists(ctx context.Context, collection string, id uint64) (bool, error)
	UpsertColumnar(ctx context.Context, collection string, ids []uint64, embeddings [][]float32, payloadJSONs []string) error
	Search(ctx context.Context, collection string, vector []float32, limit uint64) ([]string, error)
}

type VectorStoreClientSettings struct {
	Type string `json:"type"`
}

func NewVectorStoreClient(s VectorStoreClientSettings) (VectorStoreClient, error) {
	return nil, nil
}
