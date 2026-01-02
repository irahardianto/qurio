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
	properties := map[string]interface{}{
		"content":    chunk.Content,
		"url":        chunk.SourceURL,
		"sourceId":   chunk.SourceID,
		"chunkIndex": chunk.ChunkIndex,
	}
	
	if chunk.Type != "" {
		properties["type"] = chunk.Type
	}
	if chunk.Language != "" {
		properties["language"] = chunk.Language
	}
	if chunk.Title != "" {
		properties["title"] = chunk.Title
	}

	_, err := s.client.Data().Creator().
		WithClassName("DocumentChunk").
		WithProperties(properties).
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

func (s *Store) DeleteChunksBySourceID(ctx context.Context, sourceID string) error {
	_, err := s.client.Batch().ObjectsBatchDeleter().
		WithClassName("DocumentChunk").
		WithOutput("minimal").
		WithWhere(filters.Where().
			WithPath([]string{"sourceId"}).
			WithOperator(filters.Equal).
			WithValueString(sourceID)).
		Do(ctx)
	return err
}

func (s *Store) Search(ctx context.Context, query string, vector []float32, alpha float32, limit int, searchFilters map[string]interface{}) ([]retrieval.SearchResult, error) {
	hybrid := s.client.GraphQL().HybridArgumentBuilder().
		WithQuery(query).
		WithVector(vector).
		WithAlpha(alpha)

	fields := []graphql.Field{
		{Name: "content"},
		{Name: "url"},
		{Name: "sourceId"},
		{Name: "chunkIndex"},
		{Name: "type"},
		{Name: "language"},
		{Name: "title"},
		{Name: "_additional", Fields: []graphql.Field{{Name: "score"}}},
	}

	queryBuilder := s.client.GraphQL().Get().
		WithClassName("DocumentChunk").
		WithHybrid(hybrid).
		WithLimit(limit).
		WithFields(fields...)

	if len(searchFilters) > 0 {
		operands := []*filters.WhereBuilder{}
		for k, v := range searchFilters {
			if sVal, ok := v.(string); ok {
				operands = append(operands, filters.Where().
					WithPath([]string{k}).
					WithOperator(filters.Equal).
					WithValueString(sVal))
			}
		}
		
		if len(operands) > 0 {
			where := filters.Where().
				WithOperator(filters.And).
				WithOperands(operands)
			queryBuilder = queryBuilder.WithWhere(where)
		}
	}

	res, err := queryBuilder.Do(ctx)
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
					if typeVal, ok := props["type"].(string); ok {
						result.Metadata["type"] = typeVal
					}
					if langVal, ok := props["language"].(string); ok {
						result.Metadata["language"] = langVal
					}
					if titleVal, ok := props["title"].(string); ok {
						result.Metadata["title"] = titleVal
					}
					
					// Extract score
					if additional, ok := props["_additional"].(map[string]interface{}); ok {
						if score, ok := additional["score"].(string); ok {
							var fScore float64
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
		{Name: "type"},
		{Name: "language"},
		{Name: "title"},
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
					if t, ok := props["type"].(string); ok {
						chunk.Type = t
					}
					if l, ok := props["language"].(string); ok {
						chunk.Language = l
					}
					if title, ok := props["title"].(string); ok {
						chunk.Title = title
					}
					chunks = append(chunks, chunk)
				}
			}
		}
	}
	return chunks, nil
}

func (s *Store) GetChunksByURL(ctx context.Context, url string) ([]retrieval.SearchResult, error) {
	fields := []graphql.Field{
		{Name: "content"},
		{Name: "url"},
		{Name: "sourceId"},
		{Name: "chunkIndex"},
		{Name: "type"},
		{Name: "language"},
		{Name: "title"},
	}

	where := filters.Where().
		WithOperator(filters.Equal).
		WithPath([]string{"url"}).
		WithValueString(url)

	res, err := s.client.GraphQL().Get().
		WithClassName("DocumentChunk").
		WithWhere(where).
		WithLimit(1000). // Fetch up to 1000 chunks for a page
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
					if u, ok := props["url"].(string); ok {
						result.Metadata["url"] = u
					}
					if sourceId, ok := props["sourceId"].(string); ok {
						result.Metadata["sourceId"] = sourceId
					}
					if chunkIndex, ok := props["chunkIndex"].(float64); ok {
						result.Metadata["chunkIndex"] = int(chunkIndex)
					}
					if t, ok := props["type"].(string); ok {
						result.Metadata["type"] = t
					}
					if l, ok := props["language"].(string); ok {
						result.Metadata["language"] = l
					}
					if title, ok := props["title"].(string); ok {
						result.Metadata["title"] = title
					}
					results = append(results, result)
				}
			}
		}
	}
	return results, nil
}

func (s *Store) CountChunks(ctx context.Context) (int, error) {
	meta, err := s.client.GraphQL().Aggregate().
		WithClassName("DocumentChunk").
		WithFields(graphql.Field{
			Name: "meta",
			Fields: []graphql.Field{
				{Name: "count"},
			},
		}).
		Do(ctx)
	if err != nil {
		return 0, err
	}
	if len(meta.Errors) > 0 {
		return 0, fmt.Errorf("graphql error: %v", meta.Errors)
	}
	
	if data, ok := meta.Data["Aggregate"].(map[string]interface{}); ok {
		if chunks, ok := data["DocumentChunk"].([]interface{}); ok {
			if len(chunks) > 0 {
				if props, ok := chunks[0].(map[string]interface{}); ok {
					if metaStats, ok := props["meta"].(map[string]interface{}); ok {
						if count, ok := metaStats["count"].(float64); ok {
							return int(count), nil
						}
					}
				}
			}
		}
	}
	return 0, nil
}
