package weaviate

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"

	"qurio/apps/backend/internal/retrieval"
	"qurio/apps/backend/internal/vector"
	"qurio/apps/backend/internal/worker"
)

type Store struct {
	client *weaviate.Client
}

func NewStore(client *weaviate.Client) *Store {
	return &Store{client: client}
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	wAdapter := vector.NewWeaviateClientAdapter(s.client)
	return vector.EnsureSchema(ctx, wAdapter)
}

func (s *Store) StoreChunk(ctx context.Context, chunk worker.Chunk) error {
	slog.DebugContext(ctx, "storing chunk", "source_id", chunk.SourceID, "chunk_index", chunk.ChunkIndex, "url", chunk.SourceURL)
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
	if chunk.SourceName != "" {
		properties["sourceName"] = chunk.SourceName
	}
	if chunk.Author != "" {
		properties["author"] = chunk.Author
	}
	if chunk.CreatedAt != "" {
		properties["createdAt"] = chunk.CreatedAt
	}
	if chunk.PageCount > 0 {
		properties["pageCount"] = chunk.PageCount
	}

	_, err := s.client.Data().Creator().
		WithClassName("DocumentChunk").
		WithProperties(properties).
		WithVector(chunk.Vector).
		Do(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to store chunk", "error", err, "source_id", chunk.SourceID, "chunk_index", chunk.ChunkIndex)
	}
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
	slog.DebugContext(ctx, "searching vector store", "query", query, "alpha", alpha, "limit", limit)
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
		{Name: "sourceName"},
		{Name: "author"},
		{Name: "createdAt"},
		{Name: "pageCount"},
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
		slog.ErrorContext(ctx, "search failed", "error", err)
		return nil, err
	}

	if len(res.Errors) > 0 {
		msg := ""
		for _, e := range res.Errors {
			msg += fmt.Sprintf("%s; ", e.Message)
		}
		return nil, fmt.Errorf("graphql error: %s", msg)
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
						result.URL = url
						result.Metadata["url"] = url
					}
					if sourceId, ok := props["sourceId"].(string); ok {
						result.SourceID = sourceId
						result.Metadata["sourceId"] = sourceId
					}
					if chunkIndex, ok := props["chunkIndex"].(float64); ok {
						result.Metadata["chunkIndex"] = int(chunkIndex)
					}
					if typeVal, ok := props["type"].(string); ok {
						result.Type = typeVal
						result.Metadata["type"] = typeVal
					}
					if langVal, ok := props["language"].(string); ok {
						result.Language = langVal
						result.Metadata["language"] = langVal
					}
					if titleVal, ok := props["title"].(string); ok {
						result.Title = titleVal
						result.Metadata["title"] = titleVal
					}
					if sourceName, ok := props["sourceName"].(string); ok {
						result.SourceName = sourceName
						result.Metadata["sourceName"] = sourceName
					}
					if author, ok := props["author"].(string); ok {
						result.Author = author
						result.Metadata["author"] = author
					}
					if createdAt, ok := props["createdAt"].(string); ok {
						result.CreatedAt = createdAt
						result.Metadata["createdAt"] = createdAt
					}
					if pageCount, ok := props["pageCount"].(float64); ok {
						result.PageCount = int(pageCount)
						result.Metadata["pageCount"] = int(pageCount)
					}

					// Extract score
					if additional, ok := props["_additional"].(map[string]interface{}); ok {
						if score, ok := additional["score"].(string); ok {
							if fScore, err := strconv.ParseFloat(score, 64); err == nil {
								result.Score = float32(fScore)
							}
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

func (s *Store) GetChunks(ctx context.Context, sourceID string, limit, offset int) ([]worker.Chunk, error) {
	fields := []graphql.Field{
		{Name: "content"},
		{Name: "url"},
		{Name: "sourceId"},
		{Name: "chunkIndex"},
		{Name: "type"},
		{Name: "language"},
		{Name: "title"},
		{Name: "sourceName"},
	}

	where := filters.Where().
		WithOperator(filters.Equal).
		WithPath([]string{"sourceId"}).
		WithValueString(sourceID)

	res, err := s.client.GraphQL().Get().
		WithClassName("DocumentChunk").
		WithWhere(where).
		WithLimit(limit).
		WithOffset(offset).
		WithFields(fields...).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	if len(res.Errors) > 0 {
		msg := ""
		for _, e := range res.Errors {
			msg += fmt.Sprintf("%s; ", e.Message)
		}
		return nil, fmt.Errorf("graphql error: %s", msg)
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
					if sourceName, ok := props["sourceName"].(string); ok {
						chunk.SourceName = sourceName
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
		{Name: "sourceName"},
		{Name: "author"},
		{Name: "createdAt"},
		{Name: "pageCount"},
	}

	where := filters.Where().
		WithOperator(filters.Equal).
		WithPath([]string{"url"}).
		WithValueString(url)

	res, err := s.client.GraphQL().Get().
		WithClassName("DocumentChunk").
		WithWhere(where).
		WithLimit(1000). // Fetch up to 1000 chunks for a page
		WithSort(graphql.Sort{Path: []string{"chunkIndex"}, Order: graphql.Asc}).
		WithFields(fields...).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	if len(res.Errors) > 0 {
		msg := ""
		for _, e := range res.Errors {
			msg += fmt.Sprintf("%s; ", e.Message)
		}
		return nil, fmt.Errorf("graphql error: %s", msg)
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
						result.URL = u
						result.Metadata["url"] = u
					}
					if sourceId, ok := props["sourceId"].(string); ok {
						result.SourceID = sourceId
						result.Metadata["sourceId"] = sourceId
					}
					if chunkIndex, ok := props["chunkIndex"].(float64); ok {
						result.Metadata["chunkIndex"] = int(chunkIndex)
					}
					if t, ok := props["type"].(string); ok {
						result.Type = t
						result.Metadata["type"] = t
					}
					if l, ok := props["language"].(string); ok {
						result.Language = l
						result.Metadata["language"] = l
					}
					if title, ok := props["title"].(string); ok {
						result.Title = title
						result.Metadata["title"] = title
					}
					if sourceName, ok := props["sourceName"].(string); ok {
						result.SourceName = sourceName
						result.Metadata["sourceName"] = sourceName
					}
					if author, ok := props["author"].(string); ok {
						result.Author = author
						result.Metadata["author"] = author
					}
					if createdAt, ok := props["createdAt"].(string); ok {
						result.CreatedAt = createdAt
						result.Metadata["createdAt"] = createdAt
					}
					if pageCount, ok := props["pageCount"].(float64); ok {
						result.PageCount = int(pageCount)
						result.Metadata["pageCount"] = int(pageCount)
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

func (s *Store) CountChunksBySource(ctx context.Context, sourceID string) (int, error) {
	where := filters.Where().
		WithOperator(filters.Equal).
		WithPath([]string{"sourceId"}).
		WithValueString(sourceID)

	meta, err := s.client.GraphQL().Aggregate().
		WithClassName("DocumentChunk").
		WithWhere(where).
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
