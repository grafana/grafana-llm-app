package plugin

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/llm/pkg/plugin/vector/embedding"
	"github.com/grafana/llm/pkg/plugin/vector/store"
)

const (
	GrafanaDashboardsCollection = "grafana:core:dashboards"

	GrafanaDashboardsCollectionDimensions = 768
)

// embeddingClient returns an embedding client which can be used to embed text.
//
// The type of embedding engine is determined by the app's configuration.
func (app *App) embeddingClient(ctx context.Context) (embedding.EmbeddingClient, error) {
	return embedding.NewEmbeddingClient(app.embeddingSettings)
}

// vectorClient returns a vector store client which can be used to store, retrieve, and search for similar vectors.
func (app *App) vectorClient(ctx context.Context) (store.VectorStoreClient, error) {
	return store.NewVectorStoreClient(app.vectorStoreSettings)
}

// startVectorSync starts a ticker which periodically syncs Grafana metadata to the vector store.
func (app *App) startVectorSync(ctx context.Context) {
	go func() {
		log.DefaultLogger.Info("Running initial vector sync")
		if err := app.syncVectorStore(ctx); err != nil {
			log.DefaultLogger.Error("Error syncing vector store", "error", err)
		}
		log.DefaultLogger.Info("Starting vector sync ticker")
		// TODO: make sync interval configurable
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := app.syncVectorStore(ctx); err != nil {
					log.DefaultLogger.Error("Error syncing vector store", "error", err)
				}
			}
		}
	}()
}

// syncVectorStore syncs Grafana metadata to the vector store.
func (app *App) syncVectorStore(ctx context.Context) error {
	vc, err := app.vectorClient(ctx)
	if err != nil {
		return fmt.Errorf("create vector client: %w", err)
	}
	if vc == nil {
		return fmt.Errorf("vector client is nil")
	}

	ec, err := app.embeddingClient(ctx)
	if err != nil {
		return fmt.Errorf("create embedding client: %w", err)
	}
	if ec == nil {
		return fmt.Errorf("embeddings client is nil")
	}

	if err := app.syncDashboardsToVectorStore(ctx, ec, vc); err != nil {
		log.DefaultLogger.Error("Error syncing dashboards to vector store", "error", err)
		return err
	}
	return nil
}

// syncDashboardsToVectorStore syncs Grafana dashboards to the vector store.
// TODO: refactor this later to be generic over the type of metadata in some way.
func (app *App) syncDashboardsToVectorStore(ctx context.Context, ec embedding.EmbeddingClient, vc store.VectorStoreClient) error {
	log.DefaultLogger.Info("Syncing dashboards to vector store")

	if exists, err := vc.CollectionExists(ctx, GrafanaDashboardsCollection); err != nil && !exists {
		log.DefaultLogger.Info(
			"Creating dashboard collection",
			"collection", GrafanaDashboardsCollection,
			"dimensions", GrafanaDashboardsCollectionDimensions)
		err = vc.CreateCollection(ctx, GrafanaDashboardsCollection, GrafanaDashboardsCollectionDimensions)
		if err != nil {
			return fmt.Errorf("create dashboard collection: %w", err)
		}
	}

	log.DefaultLogger.Debug("Fetching dashboard list")
	// TODO: refactor all this HTTP request stuff into a Grafana client, or use a library.
	client, err := app.grafanaClient(ctx)
	if err != nil {
		return fmt.Errorf("create Grafana client: %w", err)
	}
	dashboards, err := client.Dashboards()
	if err != nil {
		return fmt.Errorf("get dashboards: %w", err)
	}
	// chunkSize is the number of dashboards to embed and add to the vector store at once.
	chunkSize := 100
	ids := make([]uint64, 0, chunkSize)
	embeddings := make([][]float32, 0, chunkSize)
	payloads := make([]string, 0, chunkSize)

	for i := 0; i < len(dashboards); i += chunkSize {
		ids = ids[:0]
		embeddings = embeddings[:0]
		payloads = payloads[:0]

		chunk := dashboards[i:int(math.Min(float64(i+chunkSize), float64(len(dashboards))))]
		for j, dashboard := range chunk {
			log.DefaultLogger.Debug("Fetching dashboard", "uid", dashboard.UID)
			// TODO: actually fetch the dashboard JSON.
			dashboardJSON := `{"dashboard": "json"}`

			// Check if dashboard exists in vector store
			hash := fnv.New64a()
			hash.Write([]byte(dashboardJSON))
			id := hash.Sum64()
			if exists, err := vc.PointExists(ctx, GrafanaDashboardsCollection, id); err != nil {
				log.DefaultLogger.Warn("check vector exists", "collection", GrafanaDashboardsCollection, "id", id, "err", err)
				continue
			} else if exists {
				log.DefaultLogger.Debug("vector already exists, skipping", "collection", GrafanaDashboardsCollection, "id", id, "err", err)
				continue
			}

			// If we're here, we have a new dashboard to embed and add.
			log.DefaultLogger.Debug("getting embeddings for dashboard", "collection", GrafanaDashboardsCollection, "index", i+j, "count", len(dashboards))
			e, err := ec.Embeddings(ctx, dashboardJSON)
			if err != nil {
				log.DefaultLogger.Warn("get embeddings", "collection", GrafanaDashboardsCollection, "err", err)
				continue
			}
			ids = append(ids, id)
			embeddings = append(embeddings, e)
			payloads = append(payloads, dashboardJSON)
		}
		if len(ids) == 0 {
			log.DefaultLogger.Debug("no new embeddings to add")
			return nil
		}
		log.DefaultLogger.Debug("adding embeddings to vector DB", "collection", GrafanaDashboardsCollection, "count", len(embeddings))
		err := vc.UpsertColumnar(ctx, GrafanaDashboardsCollection, ids, embeddings, payloads)
		if err != nil {
			return fmt.Errorf("upsert columnar: %w", err)
		}
	}
	return nil
}
