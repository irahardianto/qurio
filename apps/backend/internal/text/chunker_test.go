package text

import (
	"fmt"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestChunkMarkdown_CodeBlockPreservation(t *testing.T) {
	input := `# Header
Some prose.

` + "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```" + `

More prose.
`

	chunks := ChunkMarkdown(input, 100, 0) // Small size, but code block should stay intact
	
	foundCode := false
	for _, c := range chunks {
		if c.Type == ChunkTypeCode {
			foundCode = true
			assert.Equal(t, "go", c.Language)
			assert.Contains(t, c.Content, "func main()")
		}
	}
	assert.True(t, foundCode, "Should detect code block")
}

func TestChunkMarkdown_HeaderPreservation(t *testing.T) {
	input := `# Section 1
This is paragraph 1.

This is paragraph 2.

# Section 2
This is another section.
`
	// Use large enough maxTokens so we don't force split
	chunks := ChunkMarkdown(input, 100, 0)

	fmt.Printf("Chunks found: %d\n", len(chunks))
	for i, c := range chunks {
		fmt.Printf("Chunk %d: %q\n", i, c.Content)
	}
	
	assert.GreaterOrEqual(t, len(chunks), 2)
	if len(chunks) >= 2 {
		assert.Contains(t, chunks[0].Content, "# Section 1")
		assert.Contains(t, chunks[len(chunks)-1].Content, "# Section 2")
	}
}