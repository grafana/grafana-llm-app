package store

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	qdrant "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type qdrantSettings struct {
	Address string
}

type qdrantStore struct {
	conn              *grpc.ClientConn
	collectionsClient qdrant.CollectionsClient
	pointsClient      qdrant.PointsClient
}

func newQdrantStore(s qdrantSettings) (ReadVectorStore, func(), error) {
	conn, err := grpc.DialContext(context.Background(), s.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cancel := func() {
		defer func() {
			if err := conn.Close(); err != nil {
				log.DefaultLogger.Warn("failed to close connection", "err", err)
			}
		}()
	}
	return &qdrantStore{
		conn:              conn,
		collectionsClient: qdrant.NewCollectionsClient(conn),
		pointsClient:      qdrant.NewPointsClient(conn),
	}, cancel, nil
}

func (q *qdrantStore) Collections(ctx context.Context) ([]string, error) {
	collections, err := q.collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(collections.Collections))
	for _, c := range collections.Collections {
		names = append(names, c.Name)
	}
	return names, nil
}

func (q *qdrantStore) Search(ctx context.Context, collection string, vector []float32, limit uint64) ([]SearchResult, error) {
	result, err := q.pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collection,
		Vector:         vector,
		Limit:          limit,
		// Include all payloads in the search result
		WithVectors: &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: false}},
		WithPayload: &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(result.GetResult()))
	for _, v := range result.GetResult() {
		payload := make(map[string]any, len(v.Payload))
		for k, v := range v.Payload {
			payload[k] = v
		}
		// TODO: handle non-strings, in case they get there
		results = append(results, SearchResult{
			Score:   float64(v.Score),
			Payload: payload,
		})
	}
	return results, nil
}
