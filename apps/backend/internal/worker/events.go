package worker

type IngestEmbedPayload struct {
	SourceID      string `json:"source_id"`
	SourceURL     string `json:"source_url"`
	SourceName    string `json:"source_name"`
	Title         string `json:"title"`
	Path          string `json:"path"`

	// Chunk Data
	Content    string `json:"content"`
	ChunkIndex int    `json:"chunk_index"`
	ChunkType  string `json:"chunk_type"`
	Language   string `json:"language"`

	// Context Metadata
	Author    string `json:"author,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	PageCount int    `json:"page_count,omitempty"`

	CorrelationID string `json:"correlation_id"`
}
