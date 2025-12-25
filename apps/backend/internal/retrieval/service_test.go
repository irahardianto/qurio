package retrieval_test

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/internal/retrieval"
	"qurio/apps/backend/internal/settings"
)

type MockEmbedder struct { mock.Mock }
func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float32), args.Error(1)
}

type MockStore struct { mock.Mock }
func (m *MockStore) Search(ctx context.Context, query string, vector []float32, alpha float32, limit int) ([]retrieval.SearchResult, error) {
	args := m.Called(ctx, query, vector, alpha, limit)
	return args.Get(0).([]retrieval.SearchResult), args.Error(1)
}

type MockSettingsRepo struct { mock.Mock }
func (m *MockSettingsRepo) Get(ctx context.Context) (*settings.Settings, error) {
	args := m.Called(ctx)
	return args.Get(0).(*settings.Settings), args.Error(1)
}
func (m *MockSettingsRepo) Update(ctx context.Context, s *settings.Settings) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

type MockReranker struct { mock.Mock }
func (m *MockReranker) Rerank(ctx context.Context, query string, docs []string) ([]int, error) {
	args := m.Called(ctx, query, docs)
	return args.Get(0).([]int), args.Error(1)
}

func TestSearch_WithReranker(t *testing.T) {
	e := new(MockEmbedder)
	s := new(MockStore)
	r := new(MockReranker)
	
	repo := new(MockSettingsRepo)
	repo.On("Get", mock.Anything).Return(&settings.Settings{SearchAlpha: 0.5, SearchTopK: 10}, nil)
	setSvc := settings.NewService(repo)

	svc := retrieval.NewService(e, s, r, setSvc, nil)

	ctx := context.Background()
	e.On("Embed", ctx, "test").Return([]float32{0.1}, nil)
	
	initialResults := []retrieval.SearchResult{
		{Content: "A", Score: 0.5},
		{Content: "B", Score: 0.6},
	}
	s.On("Search", ctx, "test", []float32{0.1}, float32(0.5), 10).Return(initialResults, nil)
	
	// Reranker swaps them: [1, 0]
	r.On("Rerank", ctx, "test", []string{"A", "B"}).Return([]int{1, 0}, nil)

	res, err := svc.Search(ctx, "test", nil)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "B", res[0].Content)
	assert.Equal(t, "A", res[1].Content)
}

func TestSearch(t *testing.T) {
	e := new(MockEmbedder)
	s := new(MockStore)

	repo := new(MockSettingsRepo)
	repo.On("Get", mock.Anything).Return(&settings.Settings{SearchAlpha: 0.5, SearchTopK: 10}, nil)
	setSvc := settings.NewService(repo)

	svc := retrieval.NewService(e, s, nil, setSvc, nil)

	ctx := context.Background()
	e.On("Embed", ctx, "test").Return([]float32{0.1}, nil)
	
	expected := []retrieval.SearchResult{
		{Content: "result", Score: 0.9, Metadata: map[string]interface{}{"source": "doc1"}},
	}
	// Verify alpha is 0.5
	s.On("Search", ctx, "test", []float32{0.1}, float32(0.5), 10).Return(expected, nil)

	res, err := svc.Search(ctx, "test", nil)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "doc1", res[0].Metadata["source"])
}

func TestSearch_WithOptions(t *testing.T) {
	e := new(MockEmbedder)
	s := new(MockStore)
	
	repo := new(MockSettingsRepo)
	repo.On("Get", mock.Anything).Return(&settings.Settings{SearchAlpha: 0.5, SearchTopK: 10}, nil)
	setSvc := settings.NewService(repo)

	svc := retrieval.NewService(e, s, nil, setSvc, nil)

	ctx := context.Background()
	e.On("Embed", ctx, "test").Return([]float32{0.1}, nil)
	
	expected := []retrieval.SearchResult{}
	
	// Expect overridden alpha 0.8 and limit 5
	s.On("Search", ctx, "test", []float32{0.1}, float32(0.8), 5).Return(expected, nil)

	alpha := float32(0.8)
	limit := 5
	opts := &retrieval.SearchOptions{Alpha: &alpha, Limit: &limit}

	_, err := svc.Search(ctx, "test", opts)
	assert.NoError(t, err)
}
