package store

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	qdrant "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type qdrantSettings struct {
	// The address of the Qdrant gRPC server, e.g. localhost:6334.
	Address string `json:"address"`
	// Whether to use a secure connection.
	Secure bool `json:"secure"`
}

type qdrantStore struct {
	conn              *grpc.ClientConn
	md                *metadata.MD
	collectionsClient qdrant.CollectionsClient
	pointsClient      qdrant.PointsClient
}

func newQdrantStore(s qdrantSettings, secrets map[string]string) (VectorStore, func(), error) {
	var md *metadata.MD
	dialOptions := []grpc.DialOption{}
	if s.Secure {
		config := &tls.Config{}
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(config)))
		// Only include API key if using a secure connection.
		if key := secrets["qdrantApiKey"]; key != "" {
			meta := metadata.New(map[string]string{"api-key": key})
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

func (q *qdrantStore) Health(ctx context.Context) error {
	if q.md != nil {
		ctx = metadata.NewOutgoingContext(ctx, *q.md)
	}
	_, err := q.collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return err
	}
	return nil
}

func (q *qdrantStore) CollectionExists(ctx context.Context, collection string) (bool, error) {
	if q.md != nil {
		ctx = metadata.NewOutgoingContext(ctx, *q.md)
	}
	_, err := q.collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collection,
	}, grpc.WaitForReady(true))
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return false, err
			// Error was not a status error
		}
		if st.Code() == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (q *qdrantStore) mapFilters(ctx context.Context, filter map[string]interface{}) (*qdrant.Filter, error) {
	qdrantFilterMap := &qdrant.Filter{}

	if filter == nil {
		return qdrantFilterMap, nil
	}

	for k, v := range filter {
		switch v := v.(type) {
		case map[string]interface{}:
			for op, val := range v {
				match, err := createQdrantMatch(val)
				if err != nil {
					return nil, err
				}

				condition := &qdrant.Condition{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key:   k,
							Match: match,
						},
					},
				}

				switch op {
				case "$eq":
					qdrantFilterMap.Must = append(qdrantFilterMap.Must, condition)
				case "$ne":
					qdrantFilterMap.MustNot = append(qdrantFilterMap.MustNot, condition)
				default:
					return nil, fmt.Errorf("unsupported operator: %s", op)
				}
			}
		case []interface{}:
			switch k {
			case "$or":
				for _, u := range v {
					filterMap, err := q.mapFilters(ctx, u.(map[string]interface{}))
					if err != nil {
						return nil, err
					}
					qdrantFilterMap.Should = append(qdrantFilterMap.Should, &qdrant.Condition{
						ConditionOneOf: &qdrant.Condition_Filter{
							Filter: filterMap,
						},
					})
				}
			case "$and":
				for _, u := range v {
					filterMap, err := q.mapFilters(ctx, u.(map[string]interface{}))
					if err != nil {
						return nil, err
					}
					qdrantFilterMap.Must = append(qdrantFilterMap.Must, &qdrant.Condition{
						ConditionOneOf: &qdrant.Condition_Filter{
							Filter: filterMap,
						},
					})
				}
			default:
				return nil, fmt.Errorf("unsupported operator: %s", k)
			}
		default:
			return nil, fmt.Errorf("unsupported filter struct: %T", v)
		}
	}

	return qdrantFilterMap, nil
}

func createQdrantMatch(val interface{}) (*qdrant.Match, error) {
	match := &qdrant.Match{}
	switch val := val.(type) {
	case string:
		match.MatchValue = &qdrant.Match_Keyword{
			Keyword: val,
		}
	default:
		return nil, fmt.Errorf("unsupported filter type: %T", val)
	}
	return match, nil
}

func (q *qdrantStore) Search(ctx context.Context, collection string, vector []float32, topK uint64, filter map[string]interface{}) ([]SearchResult, error) {
	if q.md != nil {
		ctx = metadata.NewOutgoingContext(ctx, *q.md)
	}

	qdrantFilter, err := q.mapFilters(ctx, filter)
	if err != nil {
		return nil, err
	}

	result, err := q.pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collection,
		Vector:         vector,
		Limit:          topK,
		Filter:         qdrantFilter,
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
			payload[k] = fromQdrantValue(v)
		}
		// TODO: handle non-strings, in case they get there
		results = append(results, SearchResult{
			Score:   float64(v.Score),
			Payload: payload,
		})
	}
	return results, nil
}

func fromQdrantValue(in *qdrant.Value) any {
	switch v := in.Kind.(type) {
	case *qdrant.Value_NullValue:
		return nil
	case *qdrant.Value_BoolValue:
		return v.BoolValue
	case *qdrant.Value_StringValue:
		return v.StringValue
	case *qdrant.Value_IntegerValue:
		return v.IntegerValue
	case *qdrant.Value_DoubleValue:
		return v.DoubleValue
	case *qdrant.Value_ListValue:
		out := make([]any, 0, len(v.ListValue.Values))
		for _, innerV := range v.ListValue.Values {
			out = append(out, fromQdrantValue(innerV))
		}
		return out
	case *qdrant.Value_StructValue:
		out := make(map[string]any, len(v.StructValue.Fields))
		for innerK, innerV := range v.StructValue.Fields {
			out[innerK] = fromQdrantValue(innerV)
		}
		return out
	}
	return nil
}

func toQdrantValue(v reflect.Value) *qdrant.Value {
	out := &qdrant.Value{}
	switch v.Kind() {

	// Atoms
	case reflect.Invalid:
		out.Kind = &qdrant.Value_NullValue{NullValue: qdrant.NullValue_NULL_VALUE}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		out.Kind = &qdrant.Value_IntegerValue{IntegerValue: int64(v.Uint())}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		out.Kind = &qdrant.Value_IntegerValue{IntegerValue: v.Int()}
	case reflect.Float32, reflect.Float64:
		out.Kind = &qdrant.Value_DoubleValue{DoubleValue: v.Float()}
	case reflect.Bool:
		out.Kind = &qdrant.Value_BoolValue{BoolValue: v.Bool()}
	case reflect.String:
		out.Kind = &qdrant.Value_StringValue{StringValue: v.String()}

	// Slices and arrays
	case reflect.Slice, reflect.Array:
		values := make([]*qdrant.Value, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			values = append(values, toQdrantValue(v.Index(i)))
		}
		out.Kind = &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{Values: values}}

	// Maps and structs
	case reflect.Map:
		keys := v.MapKeys()
		fields := make(map[string]*qdrant.Value, len(keys))
		for _, key := range keys {
			if key.Kind() == reflect.String {
				fields[key.String()] = toQdrantValue(v.MapIndex(key))
			} else {
				log.DefaultLogger.Warn("unsupported map key type", "type", key.Kind())
			}
		}
		out.Kind = &qdrant.Value_StructValue{StructValue: &qdrant.Struct{Fields: fields}}
	case reflect.Struct:
		fields := make(map[string]*qdrant.Value, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			fields[v.Type().Field(i).Name] = toQdrantValue(v.Field(i))
		}
		out.Kind = &qdrant.Value_StructValue{StructValue: &qdrant.Struct{Fields: fields}}

		// Pointers and interfaces
	case reflect.Ptr:
		if v.IsNil() {
			out.Kind = &qdrant.Value_NullValue{NullValue: qdrant.NullValue_NULL_VALUE}
		} else {
			out = toQdrantValue(v.Elem())
		}
	case reflect.Interface:
		if v.IsNil() {
			out.Kind = &qdrant.Value_NullValue{NullValue: qdrant.NullValue_NULL_VALUE}
		} else {
			out = toQdrantValue(v.Elem())
		}
	}
	return out
}

func (q *qdrantStore) CreateCollection(ctx context.Context, collection string, size uint64) error {
	_, err := q.collectionsClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: collection,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size: size,
					// TODO: make this customizable
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	return err
}

func (q *qdrantStore) PointExists(ctx context.Context, collection string, id uint64) (bool, error) {
	point, err := q.pointsClient.Get(ctx, &qdrant.GetPoints{
		CollectionName: collection,
		Ids: []*qdrant.PointId{
			{PointIdOptions: &qdrant.PointId_Num{Num: id}},
		},
	}, grpc.WaitForReady(true))
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return false, err
			// Error was not a status error
		}
		if st.Code() == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	if point.Result == nil {
		return false, nil
	}
	return true, nil
}

func (q *qdrantStore) UpsertColumnar(ctx context.Context, collection string, ids []uint64, embeddings [][]float32, payloads []Payload) error {
	waitUpsert := false
	upsertPoints := make([]*qdrant.PointStruct, 0, len(ids))
	for i, id := range ids {
		payload := make(map[string]*qdrant.Value, len(payloads[i]))
		for k, v := range payloads[i] {
			if newV := toQdrantValue(reflect.ValueOf(v)); newV != nil {
				payload[k] = newV
			} else {
				log.DefaultLogger.Warn("unsupported payload value type", "key", k, "value", v)
			}
		}
		point := &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Num{Num: id},
			},
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: embeddings[i]}}},
			Payload: payload,
		}
		upsertPoints = append(upsertPoints, point)
	}
	_, err := q.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         upsertPoints,
		Wait:           &waitUpsert,
	}, grpc.WaitForReady(true))
	return err
}
