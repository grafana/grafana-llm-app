package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"runtime/debug"
	"time"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector/store"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const (
	GrafanaDashboardsCollection = "grafana.core.dashboards"
)

// startVectorSync starts a ticker which periodically syncs Grafana metadata to the vector store.
func (v *vectorService) StartSync() {
	go func() {
		log.DefaultLogger.Info("Running initial vector sync")
		if err := v.syncVectorStore(v.ctx); err != nil {
			log.DefaultLogger.Error("Error syncing vector store", "error", err)
		}
		log.DefaultLogger.Info("Starting vector sync ticker")
		// TODO: make sync interval configurable
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-v.ctx.Done():
				return
			case <-ticker.C:
				if err := v.syncVectorStore(v.ctx); err != nil {
					log.DefaultLogger.Error("Error syncing vector store", "error", err)
				}
			}
		}
	}()
}

// syncVectorStore syncs Grafana metadata to the vector store.
func (v *vectorService) syncVectorStore(ctx context.Context) (err error) {
	defer func() {
		if pan := recover(); pan != nil {
			err = fmt.Errorf("sync process panicked: %s %s", pan, debug.Stack())
		}
	}()
	if err = v.syncDashboardsToVectorStore(ctx); err != nil {
		log.DefaultLogger.Error("Error syncing dashboards to vector store", "error", err)
		return err
	}
	return nil
}

// syncDashboardsToVectorStore syncs Grafana dashboards to the vector store.
// TODO: refactor this later to be generic over the type of metadata in some way.
func (v *vectorService) syncDashboardsToVectorStore(ctx context.Context) error {
	log.DefaultLogger.Info("Syncing dashboards to vector store")

	if exists, err := v.store.CollectionExists(ctx, GrafanaDashboardsCollection); err == nil && !exists {
		vec, err := v.embedder.Embed(ctx, v.model, "test")
		if err != nil {
			return fmt.Errorf("embed test vector to determine dimension: %w", err)
		}
		dimension := uint64(len(vec))
		log.DefaultLogger.Info(
			"Creating dashboard collection",
			"collection", GrafanaDashboardsCollection,
			"dimensions", dimension)
		err = v.store.CreateCollection(ctx, GrafanaDashboardsCollection, dimension)
		if err != nil {
			return fmt.Errorf("create dashboard collection: %w", err)
		}
	}

	log.DefaultLogger.Info("Creating Grafana client")
	client, err := v.grafanaClient(ctx)
	if err != nil {
		return fmt.Errorf("create Grafana client: %w", err)
	}
	dashboards, err := client.Dashboards()
	if err != nil {
		log.DefaultLogger.Error("Error fetching dashboards", "err", err)
		return fmt.Errorf("get dashboards: %w", err)
	}
	log.DefaultLogger.Info("Fetched dashboards", "count", len(dashboards))
	// chunkSize is the number of dashboards to embed and add to the vector store at once.
	chunkSize := 100
	ids := make([]uint64, 0, chunkSize)
	embeddings := make([][]float32, 0, chunkSize)
	payloads := make([]store.Payload, 0, chunkSize)

	for i := 0; i < len(dashboards); i += chunkSize {
		ids = ids[:0]
		embeddings = embeddings[:0]
		payloads = payloads[:0]

		chunk := dashboards[i:int(math.Min(float64(i+chunkSize), float64(len(dashboards))))]

		for j, folderDashboard := range chunk {
			log.DefaultLogger.Debug("Fetching dashboard", "uid", folderDashboard.UID)
			dashboard, err := client.DashboardByUID(folderDashboard.UID)
			if err != nil {
				log.DefaultLogger.Warn("Unable to fetch dashboard", "uid", folderDashboard.UID, "err", err)
			}
			model := store.Payload{
				"title":       folderDashboard.Title,
				"description": dashboard.Model["description"],
			}
			// All these type assertions kinda suck, but we don't have the raw JSON model.
			// I guess we could marshal and unmarshal into a custom type?
			if panels, ok := dashboard.Model["panels"]; ok {
				if panels, ok := panels.([]any); ok {
					modelPanels := make([]map[string]any, len(panels))
					for i, panel := range panels {
						if p, ok := panel.(map[string]any); ok {
							modelPanels[i] = map[string]any{
								"title":       p["title"],
								"description": p["description"],
							}
						}
					}
					log.DefaultLogger.Info("panels", "panels", modelPanels)
					model["panels"] = modelPanels
					log.DefaultLogger.Info("model", "model", model)
				}
			}

			jdoc, err := json.Marshal(model)
			if err != nil {
				log.DefaultLogger.Warn("Unable to marshal dashboard", "uid", folderDashboard.UID, "err", err)
				continue
			}

			// Check if dashboard exists in vector store
			hash := fnv.New64a()
			hash.Write([]byte(jdoc))
			id := hash.Sum64()
			if exists, err := v.store.PointExists(ctx, GrafanaDashboardsCollection, id); err != nil {
				log.DefaultLogger.Warn("Error checking whether vector exists", "collection", GrafanaDashboardsCollection, "id", id, "err", err)
				continue
			} else if exists {
				log.DefaultLogger.Debug("Vector already exists, skipping", "collection", GrafanaDashboardsCollection, "id", id, "err", err)
				continue
			}

			// If we're here, we have a new dashboard to embed and add.
			log.DefaultLogger.Debug("Getting embeddings for dashboard", "collection", GrafanaDashboardsCollection, "index", i+j, "count", len(dashboards))
			// TODO: process the dashboard JSON.
			e, err := v.embedder.Embed(ctx, v.model, string(jdoc))
			if err != nil {
				log.DefaultLogger.Warn("Error getting embeddings", "collection", GrafanaDashboardsCollection, "err", err)
				continue
			}
			ids = append(ids, id)
			embeddings = append(embeddings, e)
			payloads = append(payloads, model)
		}
		if len(ids) == 0 {
			log.DefaultLogger.Debug("No new embeddings to add")
			return nil
		}
		log.DefaultLogger.Debug("Adding embeddings to vector DB", "collection", GrafanaDashboardsCollection, "count", len(embeddings))
		err := v.store.UpsertColumnar(ctx, GrafanaDashboardsCollection, ids, embeddings, payloads)
		if err != nil {
			return fmt.Errorf("upsert columnar: %w", err)
		}
	}
	return nil
}
