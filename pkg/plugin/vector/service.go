// package vector provides a service for searching vector embeddings.
// It combines the embedding engine and the vector store.
package vector

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/embed"
	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/oauthtokenretriever"
)

type Service interface {
	Search(ctx context.Context, collection string, query string, topK uint64, filter map[string]interface{}) ([]store.SearchResult, error)
	Health(ctx context.Context) error
	StartSync()
	Cancel()
}

type VectorSettings struct {
	Enabled   bool           `json:"enabled"`
	Model     string         `json:"model"`
	Dimension uint64         `json:"dimension"`
	Embed     embed.Settings `json:"embed"`
	Store     store.Settings `json:"store"`
}

type vectorService struct {
	embedder  embed.Embedder
	model     string
	dimension uint64
	store     store.VectorStore
	// cancel is a function to cancel the context used by the vector service
	// and/or the underlying vector store.
	cancel context.CancelFunc

	// httpClient is the http client used to make requests to the Grafana API.
	httpClient *http.Client
	// grafanaAppURL is the URL of the Grafana app. It is obtained from the
	// `GF_APP_URL` environment variable.
	grafanaAppURL string
	// tokenRetriever is used to obtain OAuth2 tokens for the vector sync process.
	tokenRetriever oauthtokenretriever.TokenRetriever
	// ctx is the context used by the vector service. It is used to obtain
	// OAuth2 tokens for the vector sync process.
	ctx context.Context
}

func NewService(s VectorSettings, secrets map[string]string, httpOpts httpclient.Options) (Service, error) {
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
	st, cancel, err := store.NewVectorStore(s.Store, secrets)
	if err != nil {
		return nil, fmt.Errorf("new vector store: %w", err)
	}
	if st == nil {
		log.DefaultLogger.Warn("No vector store configured")
		return nil, nil
	}
	v := &vectorService{
		embedder:  em,
		store:     st,
		model:     s.Model,
		dimension: s.Dimension,
		cancel:    cancel,
	}

	v.ctx = context.Background()
	v.tokenRetriever, err = oauthtokenretriever.New()
	if err != nil {
		log.DefaultLogger.Warn("Error creating token retriever, vector sync will not run", "error", err)
		return v, nil
	}

	// The Grafana URL is required to obtain tokens later on
	v.grafanaAppURL = strings.TrimRight(os.Getenv("GF_APP_URL"), "/")
	if v.grafanaAppURL == "" {
		// For debugging purposes only
		v.grafanaAppURL = "http://localhost:3000"
	}

	v.httpClient, err = httpclient.New(httpOpts)
	if err != nil {
		return nil, fmt.Errorf("httpclient new: %w", err)
	}

	v.StartSync()

	return v, nil
}

func (v *vectorService) grafanaClient(ctx context.Context) (*gapi.Client, error) {
	token, err := v.tokenRetriever.Self(ctx)
	if err != nil {
		return nil, fmt.Errorf("get OAuth token for Grafana: %w", err)
	}
	g, err := gapi.New(v.grafanaAppURL, gapi.Config{
		APIKey: token,
		Client: v.httpClient,
		// OrgID must be '1' for now.
		OrgID: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("create Grafana client: %w", err)
	}
	return g, nil
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
	return v.store.Health(ctx)
}

func (v *vectorService) Cancel() {
	if v.cancel != nil {
		v.cancel()
	}
}
