package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkMarkdown(t *testing.T) {
	t.Run("Basic Prose", func(t *testing.T) {
		text := "This is a simple paragraph."
		chunks := ChunkMarkdown(text, 100, 0)
		assert.Len(t, chunks, 1)
		assert.Equal(t, text, chunks[0].Content)
		assert.Equal(t, ChunkTypeProse, chunks[0].Type)
	})

	t.Run("Code Block", func(t *testing.T) {
		text := "Here is some code:\n```go\nfunc main() {}\n```\nEnd."
		chunks := ChunkMarkdown(text, 100, 0)
		assert.Len(t, chunks, 3)
		assert.Equal(t, "Here is some code:", chunks[0].Content)
		assert.Equal(t, "```go\nfunc main() {}\n```", chunks[1].Content)
		assert.Equal(t, "go", chunks[1].Language)
		assert.Equal(t, ChunkTypeCode, chunks[1].Type)
		assert.Equal(t, "End.", chunks[2].Content)
	})

	t.Run("Code Block Types", func(t *testing.T) {
		tests := []struct {
			lang string
			want ChunkType
		}{
			{"json", ChunkTypeConfig},
			{"bash", ChunkTypeCmd},
			{"openapi", ChunkTypeAPI},
			{"python", ChunkTypeCode},
		}

		for _, tt := range tests {
			text := "```" + tt.lang + "\ncontent\n```"
			chunks := ChunkMarkdown(text, 100, 0)
			assert.Len(t, chunks, 1)
			assert.Equal(t, tt.want, chunks[0].Type)
		}
	})

	t.Run("Large Code Block Split", func(t *testing.T) {
		// Create large content > maxTokens (approx 4 chars/token)
		// Max 10 tokens = 40 chars
		line := "1234567890" // 10 chars
		content := ""
		for i := 0; i < 10; i++ {
			content += line + "\n"
		}
		// Total ~110 chars. Max 10 tokens (40 chars)
		text := "```txt\n" + content + "```"

		chunks := ChunkMarkdown(text, 10, 0)
		assert.True(t, len(chunks) > 1)
		assert.Contains(t, chunks[0].Content, "```txt")
	})
}

func TestChunkProse(t *testing.T) {
	t.Run("Headers Split", func(t *testing.T) {
		text := "# Header 1\nContent 1\n## Header 2\nContent 2"
		chunks := chunkProse(text, 100, 0)
		assert.Len(t, chunks, 2)
		assert.Contains(t, chunks[0].Content, "Header 1")
		assert.Contains(t, chunks[1].Content, "Header 2")
	})

	t.Run("Paragraph Split", func(t *testing.T) {
		// Max 10 tokens ~ 40 chars
		para1 := "Short paragraph."
		para2 := "Another short paragraph."
		text := para1 + "\n\n" + para2

		// If maxTokens is small enough to force split
		// "Short paragraph." (16) -> Chunk 1
		// "Another short paragraph." (24) -> Split to "Another short" (13) and "paragraph." (10)
		chunks := chunkProse(text, 5, 0) // Very small limit (approx 20 chars)
		assert.Len(t, chunks, 3)
	})

	t.Run("Line Split", func(t *testing.T) {
		// Large paragraph
		line1 := "Line 1 is long enough."
		line2 := "Line 2 is also long."
		text := line1 + "\n" + line2

		chunks := chunkProse(text, 5, 0)
		assert.True(t, len(chunks) >= 2)
	})

	t.Run("Word Split", func(t *testing.T) {
		// Very long line
		text := "VeryLongWordThatExceedsLimit AnotherWord"
		chunks := chunkProse(text, 2, 0) // ~8 chars
		assert.True(t, len(chunks) >= 2)
	})
}

func TestDetectChunkType(t *testing.T) {
	assert.Equal(t, ChunkTypeAPI, detectChunkType("Swagger API Definition"))
	assert.Equal(t, ChunkTypeAPI, detectChunkType("API Endpoint URL Method"))
	assert.Equal(t, ChunkTypeProse, detectChunkType("Just some text"))
}
