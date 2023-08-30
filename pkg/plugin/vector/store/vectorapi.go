package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type grafanaVectorAPISettings struct {
	URL string `json:"url"`
}

type grafanaVectorAPI struct {
	client *http.Client
	url    string
}

func (g *grafanaVectorAPI) Collections(ctx context.Context) ([]string, error) {
	resp, err := g.client.Get(g.url + "/collections")
	if err != nil {
		return nil, fmt.Errorf("get collections: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get collections: %s", resp.Status)
	}

	type collectionResponse struct {
		Name      string `json:"name"`
		Dimension int    `json:"dimension"`
	}
	collections := []collectionResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return nil, fmt.Errorf("decode collections: %w", err)
	}
	names := make([]string, 0, len(collections))
	for _, c := range collections {
		names = append(names, c.Name)
	}
	return names, nil
}

func (g *grafanaVectorAPI) Search(ctx context.Context, collection string, vector []float32, limit uint64) ([]SearchResult, error) {
	type queryPointsRequest struct {
		Query []float32 `json:"query"`
		TopK  uint64    `json:"top_k"`
	}
	reqBody := queryPointsRequest{
		Query: vector,
		TopK:  limit,
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	resp, err := g.client.Post(g.url+"/collections/"+collection+"/query", "application/json", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("post collections: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post collections: %s", resp.Status)
	}
	type queryPointResult struct {
		ID        string         `json:"id"`
		Embedding []float32      `json:"embedding"`
		Metadata  map[string]any `json:"metadata"`
		Score     float64        `json:"score"`
	}
	queryResult := []queryPointResult{}
	if err := json.NewDecoder(resp.Body).Decode(&queryResult); err != nil {
		return nil, fmt.Errorf("decode collections: %w", err)
	}
	results := make([]SearchResult, 0, len(queryResult))
	for _, r := range queryResult {
		results = append(results, SearchResult{
			Payload: r.Metadata,
			Score:   r.Score,
		})
	}
	return results, nil
}

func newGrafanaVectorAPI(s grafanaVectorAPISettings) ReadVectorStore {
	return &grafanaVectorAPI{
		client: &http.Client{},
		url:    s.URL,
	}
}
