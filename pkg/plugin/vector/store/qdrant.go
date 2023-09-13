package store

import (
	"context"
	"crypto/tls"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	qdrant "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type qdrantSettings struct {
	// The address of the Qdrant gRPC server, e.g. localhost:6334.
	Address string `json:"address"`
	// Whether to use a secure connection.
	Secure bool `json:"secure"`
	// The API key to use for authentication. Only used when secure is true.
	APIKey string `json:"apiKey"`
}

type qdrantStore struct {
	conn              *grpc.ClientConn
	md                *metadata.MD
	collectionsClient qdrant.CollectionsClient
	pointsClient      qdrant.PointsClient
}

func newQdrantStore(s qdrantSettings) (ReadVectorStore, func(), error) {
	var md *metadata.MD
	dialOptions := []grpc.DialOption{}
	if s.Secure {
		config := &tls.Config{}
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(config)))
		// Only include API key if using a secure connection.
		if s.APIKey != "" {
			meta := metadata.New(map[string]string{"api-key": s.APIKey})
			md = &meta
		}
	} else {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.DialContext(context.Background(), s.Address, dialOptions...)
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
		md:                md,
		collectionsClient: qdrant.NewCollectionsClient(conn),
		pointsClient:      qdrant.NewPointsClient(conn),
	}, cancel, nil
}

func (q *qdrantStore) Collections(ctx context.Context) ([]string, error) {
	if q.md != nil {
		ctx = metadata.NewOutgoingContext(ctx, *q.md)
	}
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
	if q.md != nil {
		ctx = metadata.NewOutgoingContext(ctx, *q.md)
	}
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
