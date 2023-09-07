// package vector provides a service for searching vector embeddings.
// It combines the embedding engine and the vector store.
package vector

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/llm/pkg/plugin/vector/embed"
	"github.com/grafana/llm/pkg/plugin/vector/store"
)

type Service interface {
	Search(ctx context.Context, collection string, query string, limit uint64) ([]store.SearchResult, error)
}

type vectorService struct {
	embedder    embed.Embedder
	store       store.ReadVectorStore
	collections map[string]store.Collection
}

func NewService(embedSettings embed.Settings, storeSettings store.Settings) (Service, error) {
	log.DefaultLogger.Debug("Creating embedder")
	em, err := embed.NewEmbedder(embedSettings)
	if err != nil {
		return nil, fmt.Errorf("new embedder: %w", err)
	}
	if em == nil {
		log.DefaultLogger.Warn("No embedder configured")
		return nil, nil
	}
	log.DefaultLogger.Info("Creating vector store")
	st, err := store.NewReadVectorStore(storeSettings)
	if err != nil {
		return nil, fmt.Errorf("new vector store: %w", err)
	}
	if st == nil {
		log.DefaultLogger.Warn("No vector store configured")
		return nil, nil
	}
	collections := make(map[string]store.Collection, len(storeSettings.Collections))
	for _, c := range storeSettings.Collections {
		collections[c.Name] = c
	}
	return &vectorService{
		embedder:    em,
		store:       st,
		collections: collections,
	}, nil
}

func (g vectorService) Search(ctx context.Context, collection string, query string, limit uint64) ([]store.SearchResult, error) {
	// Determine which model was used to embed this collection.
	c := g.collections[collection]
	if c.Name == "" {
		return nil, fmt.Errorf("unknown collection %s", collection)
	}

	log.DefaultLogger.Info("Embedding", "model", c.Model, "query", query)
	// Get the embedding for the search query.
	e, err := g.embedder.Embed(ctx, c.Model, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	log.DefaultLogger.Info("Searching", "collection", collection, "query", query)
	// Search the vector store for similar vectors.
	results, err := g.store.Search(ctx, collection, e, limit)
	if err != nil {
		return nil, fmt.Errorf("vector store search: %w", err)
	}

	return results, nil
}
