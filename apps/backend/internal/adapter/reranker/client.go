package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	apiKey   string
	provider string
	client   *http.Client
	baseURL  string
}

func NewClient(provider, apiKey string) *Client {
	return &Client{
		provider: provider,
		apiKey:   apiKey,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

func (c *Client) Rerank(ctx context.Context, query string, docs []string) ([]int, error) {
	if c.provider == "jina" {
		return c.rerankJina(ctx, query, docs)
	}
	if c.provider == "cohere" {
		return c.rerankCohere(ctx, query, docs)
	}
	// Return identity indices
	indices := make([]int, len(docs))
	for i := range indices {
		indices[i] = i
	}
	return indices, nil
}

func (c *Client) rerankJina(ctx context.Context, query string, docs []string) ([]int, error) {
	url := "https://api.jina.ai/v1/rerank"
	if c.baseURL != "" {
		url = c.baseURL
	}

	reqBody := map[string]interface{}{
		"model":     "jina-reranker-v1-base-en",
		"query":     query,
		"documents": docs,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jina api error: %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			Index int     `json:"index"`
			Score float64 `json:"relevance_score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	indices := make([]int, 0, len(docs))
	for _, r := range result.Results {
		if r.Index < len(docs) {
			indices = append(indices, r.Index)
		}
	}
	
	return indices, nil
}

func (c *Client) rerankCohere(ctx context.Context, query string, docs []string) ([]int, error) {
	url := "https://api.cohere.ai/v1/rerank"
	if c.baseURL != "" {
		url = c.baseURL
	}

	reqBody := map[string]interface{}{
		"model":            "rerank-english-v3.0",
		"query":            query,
		"documents":        docs,
		"top_n":            len(docs),
		"return_documents": false,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("cohere api error: %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			Index int     `json:"index"`
			Score float64 `json:"relevance_score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	indices := make([]int, 0, len(docs))
	for _, r := range result.Results {
		if r.Index < len(docs) {
			indices = append(indices, r.Index)
		}
	}

	return indices, nil
}