package weaviate

import (
	"context"
	"fmt"
	"qurio/apps/backend/internal/retrieval"
	"qurio/apps/backend/internal/worker"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
)

type Store struct {
	client *weaviate.Client
}

func NewStore(client *weaviate.Client) *Store {
	return &Store{client: client}
}

func (s *Store) StoreChunk(ctx context.Context, chunk worker.Chunk) error {
	_, err := s.client.Data().Creator().
		WithClassName("DocumentChunk").
		WithProperties(map[string]interface{}{
			"content":    chunk.Content,
			"url":        chunk.SourceURL,
			"sourceId":   chunk.SourceID,
			"chunkIndex": chunk.ChunkIndex,
		}).
		WithVector(chunk.Vector).
		Do(ctx)
	return err
}

func (s *Store) DeleteChunksByURL(ctx context.Context, sourceID, url string) error {
	_, err := s.client.Batch().ObjectsBatchDeleter().
		WithClassName("DocumentChunk").
		WithOutput("minimal").
		WithWhere(filters.Where().
			WithOperator(filters.And).
			WithOperands([]*filters.WhereBuilder{
				filters.Where().
					WithPath([]string{"sourceId"}).
					WithOperator(filters.Equal).
					WithValueString(sourceID),
				filters.Where().
					WithPath([]string{"url"}).
					WithOperator(filters.Equal).
					WithValueString(url),
			})).
		Do(ctx)
	return err
}

func (s *Store) Search(ctx context.Context, query string, vector []float32, alpha float32, limit int) ([]retrieval.SearchResult, error) {
	hybrid := s.client.GraphQL().HybridArgumentBuilder().
		WithQuery(query).
		WithVector(vector).
		WithAlpha(alpha)

	fields := []graphql.Field{
		{Name: "content"},
		{Name: "url"},
		{Name: "sourceId"},
		{Name: "chunkIndex"},
		{Name: "_additional", Fields: []graphql.Field{{Name: "score"}}},
	}

	res, err := s.client.GraphQL().Get().
		WithClassName("DocumentChunk").
		WithHybrid(hybrid).
		WithLimit(limit).
		WithFields(fields...).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %v", res.Errors)
	}

	var results []retrieval.SearchResult
	if data, ok := res.Data["Get"].(map[string]interface{}); ok {
		if chunks, ok := data["DocumentChunk"].([]interface{}); ok {
			for _, c := range chunks {
				if props, ok := c.(map[string]interface{}); ok {
					result := retrieval.SearchResult{
						Metadata: make(map[string]interface{}),
					}
					
					if content, ok := props["content"].(string); ok {
						result.Content = content
					}
					if url, ok := props["url"].(string); ok {
						result.Metadata["url"] = url
					}
					if sourceId, ok := props["sourceId"].(string); ok {
						result.Metadata["sourceId"] = sourceId
					}
					if chunkIndex, ok := props["chunkIndex"].(float64); ok {
						result.Metadata["chunkIndex"] = int(chunkIndex)
					}
					
					// Extract score
					if additional, ok := props["_additional"].(map[string]interface{}); ok {
						if score, ok := additional["score"].(string); ok {
							// Weaviate returns score as string in some versions, or float in others.
							// Assuming string or handling both might be safer, but let's try assuming float first or converting.
							// Wait, the go client might return it as string.
							// Let's print type if needed, but standard JSON usually decodes numbers as float64.
							// However, GraphQL additional fields can be tricky.
							// Let's assume float64 for now, if not check string.
							var fScore float64
							// It often comes as string in additional
							fmt.Sscanf(score, "%f", &fScore)
							result.Score = float32(fScore)
						} else if score, ok := additional["score"].(float64); ok {
							result.Score = float32(score)
						}
					}

					results = append(results, result)
				}
			}
		}
	}

	return results, nil
}

func (s *Store) GetChunks(ctx context.Context, sourceID string) ([]worker.Chunk, error) {
	fields := []graphql.Field{
		{Name: "content"},
		{Name: "url"},
		{Name: "sourceId"},
		{Name: "chunkIndex"},
	}

	where := filters.Where().
		WithOperator(filters.Equal).
		WithPath([]string{"sourceId"}).
		WithValueString(sourceID)

	res, err := s.client.GraphQL().Get().
		WithClassName("DocumentChunk").
		WithWhere(where).
		WithLimit(100).
		WithFields(fields...).
		Do(ctx)
	
	if err != nil {
		return nil, err
	}
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %v", res.Errors)
	}

	var chunks []worker.Chunk
	if data, ok := res.Data["Get"].(map[string]interface{}); ok {
		if rawChunks, ok := data["DocumentChunk"].([]interface{}); ok {
			for _, c := range rawChunks {
				if props, ok := c.(map[string]interface{}); ok {
					chunk := worker.Chunk{}
					if content, ok := props["content"].(string); ok {
						chunk.Content = content
					}
					if url, ok := props["url"].(string); ok {
						chunk.SourceURL = url
					}
					if sID, ok := props["sourceId"].(string); ok {
						chunk.SourceID = sID
					}
					if idx, ok := props["chunkIndex"].(float64); ok {
						chunk.ChunkIndex = int(idx)
					}
					chunks = append(chunks, chunk)
				}
			}
		}
	}
	return chunks, nil
}
