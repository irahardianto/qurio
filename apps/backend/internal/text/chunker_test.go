package text

import (
	"strings"
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
		// "End." is filtered as noise (single short word)
		// "Here is some code:" may also be filtered (short label, <=3 words)
		assert.True(t, len(chunks) >= 1, "should have at least the code chunk")
		// Find the code chunk
		var codeChunk *ChunkResult
		for i := range chunks {
			if chunks[i].Type == ChunkTypeCode {
				codeChunk = &chunks[i]
			}
		}
		assert.NotNil(t, codeChunk, "should have a code chunk")
		assert.Equal(t, "```go\nfunc main() {}\n```", codeChunk.Content)
		assert.Equal(t, "go", codeChunk.Language)
		assert.Equal(t, ChunkTypeCode, codeChunk.Type)
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

func TestIsNoiseChunk(t *testing.T) {
	t.Run("Empty content is noise", func(t *testing.T) {
		assert.True(t, IsNoiseChunk(""))
		assert.True(t, IsNoiseChunk("   "))
	})

	t.Run("Install commands are noise", func(t *testing.T) {
		assert.True(t, IsNoiseChunk("npm install shadcn-vue"))
		assert.True(t, IsNoiseChunk("pnpm add @tanstack/vue-query"))
		assert.True(t, IsNoiseChunk("yarn add react"))
		assert.True(t, IsNoiseChunk("pip install django"))
		assert.True(t, IsNoiseChunk("cargo add serde"))
		assert.True(t, IsNoiseChunk("go get github.com/gin-gonic/gin"))
	})

	t.Run("Install with explanation is NOT noise", func(t *testing.T) {
		content := "To get started with React Query, install the package:\n\nnpm install @tanstack/react-query\n\nThen wrap your app in the QueryClientProvider."
		assert.False(t, IsNoiseChunk(content))
	})

	t.Run("Navigation link lists are noise", func(t *testing.T) {
		content := "[Home](/)\n[About](/about)\n[Docs](/docs)\n[API](/api)\n[Blog](/blog)"
		assert.True(t, IsNoiseChunk(content))
	})

	t.Run("Content with some links is NOT noise", func(t *testing.T) {
		content := "## Related Resources\n\nFor more information on routing, see the [Vue Router docs](https://router.vuejs.org).\n\nThe middleware pattern is explained in [Express guide](https://expressjs.com)."
		assert.False(t, IsNoiseChunk(content))
	})

	t.Run("Short labels are noise", func(t *testing.T) {
		assert.True(t, IsNoiseChunk("Overview"))
		assert.True(t, IsNoiseChunk("Getting Started"))
		assert.True(t, IsNoiseChunk("# API"))
	})

	t.Run("Short code snippet is NOT noise", func(t *testing.T) {
		assert.False(t, IsNoiseChunk("```go\nfmt.Println()\n```"))
	})

	t.Run("Copyright is noise when short", func(t *testing.T) {
		assert.True(t, IsNoiseChunk("© 2024 Example Corp. All rights reserved."))
		assert.True(t, IsNoiseChunk("Terms of Service | Privacy Policy"))
	})

	t.Run("Real documentation content is NOT noise", func(t *testing.T) {
		content := "## useQuery Hook\n\nThe useQuery hook is the primary way to fetch data in React Query. It accepts a query key and a query function."
		assert.False(t, IsNoiseChunk(content))
	})

	t.Run("Code explanation with imports is NOT noise", func(t *testing.T) {
		content := "Import the createApp function from Vue and mount your application to the DOM element."
		assert.False(t, IsNoiseChunk(content))
	})
}

func TestCleanMarkdownNoise(t *testing.T) {
	t.Run("Strips edit links", func(t *testing.T) {
		input := "Some content\n[Edit this page](https://github.com/edit)\nMore content"
		result := CleanMarkdownNoise(input)
		assert.NotContains(t, result, "Edit this page")
		assert.Contains(t, result, "Some content")
		assert.Contains(t, result, "More content")
	})

	t.Run("Strips table of contents", func(t *testing.T) {
		input := "## Table of Contents\n- [Section 1](#section-1)\n- [Section 2](#section-2)\n\n## Section 1\nReal content here"
		result := CleanMarkdownNoise(input)
		assert.NotContains(t, result, "Table of Contents")
		assert.Contains(t, result, "Section 1")
		assert.Contains(t, result, "Real content here")
	})

	t.Run("Preserves normal content", func(t *testing.T) {
		input := "# API Reference\n\nThe `createApp` function initializes a new Vue application instance."
		result := CleanMarkdownNoise(input)
		assert.Equal(t, input, result)
	})
}

func TestChunkMarkdown_NoiseFiltering(t *testing.T) {
	t.Run("Filters install-only chunks", func(t *testing.T) {
		text := "# Getting Started\n\nThis is a guide.\n\nnpm install my-package\n\n## Next Steps\n\nConfigure your application by editing the config file."
		chunks := ChunkMarkdown(text, 100, 0)
		for _, c := range chunks {
			assert.NotEqual(t, "npm install my-package", strings.TrimSpace(c.Content))
		}
	})

	t.Run("Keeps code blocks even with install commands", func(t *testing.T) {
		text := "Install the package:\n```bash\nnpm install my-package\n```\nThen configure it."
		chunks := ChunkMarkdown(text, 100, 0)
		hasCodeBlock := false
		for _, c := range chunks {
			if c.Type == ChunkTypeCmd {
				hasCodeBlock = true
			}
		}
		assert.True(t, hasCodeBlock, "Code block with install command should be preserved")
	})
}
