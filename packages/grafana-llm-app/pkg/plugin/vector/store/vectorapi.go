package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type GrafanaVectorAPISettings struct {
	URL           string `json:"url"`
	AuthType      string `json:"authType"`
	BasicAuthUser string `json:"basicAuthUser"`
}

type grafanaVectorAPIAuthSettings struct {
	BasicAuthUser     string
	BasicAuthPassword string
}

type grafanaVectorAPI struct {
	client       *http.Client
	url          string
	authType     VectorStoreAuthType
	authSettings grafanaVectorAPIAuthSettings
}

func (g *grafanaVectorAPI) setAuth(req *http.Request) {
	switch g.authType {
	case VectorStoreAuthTypeBasicAuth:
		req.SetBasicAuth(g.authSettings.BasicAuthUser, g.authSettings.BasicAuthPassword)
	}
}

func (g *grafanaVectorAPI) CollectionExists(ctx context.Context, collection string) (bool, error) {
	req, err := http.NewRequest("GET", g.url+"/v1/collections/"+collection, nil)
	if err != nil {
		return false, fmt.Errorf("get collection: %w", err)
	}
	g.setAuth(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("get collection: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("get collection: %s", resp.Status)
	}
	return true, nil
}

func (g *grafanaVectorAPI) Search(ctx context.Context, collection string, vector []float32, topK uint64, filter map[string]interface{}) ([]SearchResult, error) {
	type queryPointsRequest struct {
		Query []float32 `json:"query"`
		TopK  uint64    `json:"top_k"`
		// optional filter json field
		Filter map[string]interface{} `json:"filter"`
	}
	reqBody := queryPointsRequest{
		Query:  vector,
		TopK:   topK,
		Filter: filter,
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", g.url+"/v1/collections/"+collection+"/query", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("get collections: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	g.setAuth(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post collections: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.DefaultLogger.Warn("failed to close response body", "err", err)
		}
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post collections: %s", resp.Status)
	}
	type queryPointPayload struct {
		ID        string         `json:"id"`
		Embedding []float32      `json:"embedding"`
		Metadata  map[string]any `json:"metadata"`
	}
	type queryPointResult struct {
		Payload queryPointPayload `json:"payload"`
		Score   float64           `json:"score"`
	}
	queryResult := []queryPointResult{}
	if err := json.Unmarshal(body, &queryResult); err != nil {
		return nil, fmt.Errorf("decode collections: %w", err)
	}
	results := make([]SearchResult, 0, len(queryResult))
	for _, r := range queryResult {
		results = append(results, SearchResult{
			Payload: r.Payload.Metadata,
			Score:   r.Score,
		})
	}
	return results, nil
}

func (g *grafanaVectorAPI) Health(ctx context.Context) error {
	req, err := http.NewRequest("GET", g.url+"/healthz", nil)
	if err != nil {
		return fmt.Errorf("get health: %w", err)
	}
	g.setAuth(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("get health: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get health: %s", resp.Status)
	}
	return nil
}

func newGrafanaVectorAPI(s GrafanaVectorAPISettings, secrets map[string]string) (ReadVectorStore, error) {
	return &grafanaVectorAPI{
		client:   &http.Client{},
		url:      s.URL,
		authType: VectorStoreAuthType(s.AuthType),
		authSettings: grafanaVectorAPIAuthSettings{
			BasicAuthUser:     s.BasicAuthUser,
			BasicAuthPassword: secrets["vectorStoreBasicAuthPassword"],
		},
	}, nil
}
