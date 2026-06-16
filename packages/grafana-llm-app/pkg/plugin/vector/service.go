// package vector provides a service for searching vector embeddings.
// It combines the embedding engine and the vector store.
package vector

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/embed"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Service interface {
	Search(ctx context.Context, collection string, query string, topK uint64, filter map[string]interface{}) ([]store.SearchResult, error)
	Health(ctx context.Context) error
	Cancel()
}

type VectorSettings struct {
	Enabled bool           `json:"enabled"`
	Model   string         `json:"model"`
	Embed   embed.Settings `json:"embed"`
	Store   store.Settings `json:"store"`
}

type vectorService struct {
	embedder embed.Embedder
	model    string
	store    store.ReadVectorStore
	cancel   context.CancelFunc
}

func NewService(s VectorSettings, secrets map[string]string) (Service, error) {
	log.DefaultLogger.Debug("Creating embedder")
	em, err := embed.NewEmbedder(s.Embed, secrets)
	if err != nil {
		return nil, fmt.Errorf("new embedder: %w", err)
	}
	if em == nil {
		log.DefaultLogger.Warn("No embedder configured")
		return nil, nil
	}
	log.DefaultLogger.Info("Creating vector store")
	st, cancel, err := store.NewReadVectorStore(s.Store, secrets)
	if err != nil {
		return nil, fmt.Errorf("new vector store: %w", err)
	}
	if st == nil {
		log.DefaultLogger.Warn("No vector store configured")
		return nil, nil
	}

	return &vectorService{
		embedder: em,
		store:    st,
		model:    s.Model,
		cancel:   cancel,
	}, nil
}

func (v *vectorService) Search(ctx context.Context, collection string, query string, topK uint64, filter map[string]interface{}) ([]store.SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	exists, err := v.store.CollectionExists(ctx, collection)
	if err != nil {
		return nil, fmt.Errorf("vector store collections: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("collection %s not found in store", collection)
	}

	log.DefaultLogger.Info("Embedding", "model", v.model, "query", query)
	// Get the embedding for the search query.
	e, err := v.embedder.Embed(ctx, v.model, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	log.DefaultLogger.Info("Searching", "collection", collection, "query", query)
	// Search the vector store for similar vectors.
	results, err := v.store.Search(ctx, collection, e, topK, filter)
	if err != nil {
		return nil, fmt.Errorf("vector store search: %w", err)
	}

	return results, nil
}

func (v *vectorService) Health(ctx context.Context) error {
	err := v.store.Health(ctx)
	if err != nil {
		return fmt.Errorf("vector store health: %w", err)
	}
	err = v.embedder.Health(ctx, v.model)
	if err != nil {
		return fmt.Errorf("embedder health: %w", err)
	}
	return nil
}

func (v vectorService) Cancel() {
	if v.cancel != nil {
		v.cancel()
	}
}
