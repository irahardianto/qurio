package text

import (
	"regexp"
	"strings"
)

type ChunkType string

const (
	ChunkTypeProse  ChunkType = "prose"
	ChunkTypeCode   ChunkType = "code"
	ChunkTypeAPI    ChunkType = "api"
	ChunkTypeConfig ChunkType = "config"
	ChunkTypeCmd    ChunkType = "cmd"
)

type ChunkResult struct {
	Content  string
	Type     ChunkType
	Language string
}

// ChunkMarkdown implements a simplified chunker that splits text into chunks,
// preserving code blocks and identifying their language.
// It also splits large prose blocks into smaller chunks.
func ChunkMarkdown(text string, maxTokens, overlap int) []ChunkResult {
	var results []ChunkResult

	// Regex for code fences: ```lang\n content \n```
	// We use (?s) to allow . to match newlines
	// Safe regex using character classes to avoid escape hell in Go strings
	re := regexp.MustCompile("(?s)```([a-zA-Z0-9_]+)?[[:space:]]*\\n(.*?)\\n[[:space:]]*```")

	lastIndex := 0
	matches := re.FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		// 1. Prose before the code block
		if match[0] > lastIndex {
			prose := strings.TrimSpace(text[lastIndex:match[0]])
			if len(prose) > 0 {
				proseChunks := chunkProse(prose, maxTokens, overlap)
				results = append(results, proseChunks...)
			}
		}

		// 2. The code block itself
		lang := ""
		if match[2] != -1 {
			lang = text[match[2]:match[3]]
		}
		content := text[match[4]:match[5]]

		cType := ChunkTypeCode
		if lang == "yaml" || lang == "json" || lang == "toml" {
			cType = ChunkTypeConfig
		} else if lang == "bash" || lang == "sh" || lang == "shell" {
			cType = ChunkTypeCmd
		}

		// Estimate tokens (approx 4 chars per token)
		estimatedTokens := len(content) / 4
		if estimatedTokens > maxTokens {
			codeChunks := chunkCode(content, lang, cType, maxTokens)
			results = append(results, codeChunks...)
		} else {
			fullBlock := "```" + lang + "\n" + content + "\n```"
			results = append(results, ChunkResult{
				Content:  fullBlock,
				Type:     cType,
				Language: lang,
			})
		}

		lastIndex = match[1]
	}

	// 3. Remaining prose after the last code block
	if lastIndex < len(text) {
		prose := strings.TrimSpace(text[lastIndex:])
		if len(prose) > 0 {
			proseChunks := chunkProse(prose, maxTokens, overlap)
			results = append(results, proseChunks...)
		}
	}

	return results
}

// chunkProse splits prose into chunks respecting structure: Headers -> Paragraphs -> Lines -> Words
func chunkProse(text string, maxTokens, overlap int) []ChunkResult {
	if text == "" {
		return nil
	}
	
	// Approx chars per token
	maxChars := maxTokens * 4
	
	// 1. Split by Headers (level 1-6)
	headerRe := regexp.MustCompile(`(?m)^#{1,6}\s`)
	headerIndices := headerRe.FindAllStringIndex(text, -1)
	
	var sections []string
	lastIdx := 0
	
	for _, loc := range headerIndices {
		if loc[0] > lastIdx {
			sections = append(sections, text[lastIdx:loc[0]])
		}
		lastIdx = loc[0]
	}
	if lastIdx < len(text) {
		sections = append(sections, text[lastIdx:])
	}
	
	var chunks []ChunkResult
	
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if len(section) == 0 {
			continue
		}
		
		if len(section) <= maxChars {
			chunks = append(chunks, ChunkResult{Content: section, Type: ChunkTypeProse})
			continue
		}
		
		// 2. Split by Paragraphs
		paragraphs := strings.Split(section, "\n\n")
		var currentChunk strings.Builder
		
		for _, para := range paragraphs {
			para = strings.TrimSpace(para)
			if len(para) == 0 {
				continue
			}
		
			// If paragraph fits in current chunk
			if currentChunk.Len() + len(para) + 2 <= maxChars {
				if currentChunk.Len() > 0 {
					currentChunk.WriteString("\n\n")
				}
				currentChunk.WriteString(para)
			} else {
				// Flush current chunk if not empty
				if currentChunk.Len() > 0 {
					chunks = append(chunks, ChunkResult{Content: currentChunk.String(), Type: ChunkTypeProse})
					currentChunk.Reset()
				}
				
				// Handle large paragraph
				if len(para) > maxChars {
					// 3. Split by Lines
					lines := strings.Split(para, "\n")
					for _, line := range lines {
						if currentChunk.Len() + len(line) + 1 <= maxChars {
							if currentChunk.Len() > 0 {
								currentChunk.WriteString("\n")
							}
							currentChunk.WriteString(line)
						} else {
							if currentChunk.Len() > 0 {
								chunks = append(chunks, ChunkResult{Content: currentChunk.String(), Type: ChunkTypeProse})
								currentChunk.Reset()
							}
							
							// 4. Split by Words (Fallback)
							if len(line) > maxChars {
								words := strings.Fields(line)
								for _, word := range words {
									if currentChunk.Len() + len(word) + 1 <= maxChars {
										if currentChunk.Len() > 0 {
											currentChunk.WriteString(" ")
										}
										currentChunk.WriteString(word)
									} else {
										chunks = append(chunks, ChunkResult{Content: currentChunk.String(), Type: ChunkTypeProse})
										currentChunk.Reset()
										currentChunk.WriteString(word)
									}
								}
							} else {
								currentChunk.WriteString(line)
							}
						}
					}
				} else {
					currentChunk.WriteString(para)
				}
			}
		}
		
		if currentChunk.Len() > 0 {
			chunks = append(chunks, ChunkResult{Content: currentChunk.String(), Type: ChunkTypeProse})
		}
	}
	
	return chunks
}

// chunkCode splits a large code block into smaller chunks by line
func chunkCode(content, lang string, cType ChunkType, maxTokens int) []ChunkResult {
	lines := strings.Split(content, "\n")
	var chunks []ChunkResult
	
	charsPerToken := 4
	maxChars := maxTokens * charsPerToken
	
	var currentChunk strings.Builder
	currentLen := 0
	
	for _, line := range lines {
		lineLen := len(line) + 1 
		
		if currentLen + lineLen > maxChars && currentLen > 0 {
			chunks = append(chunks, ChunkResult{
				Content:  "```" + lang + "\n" + currentChunk.String() + "\n```",
				Type:     cType,
				Language: lang,
			})
			currentChunk.Reset()
			currentLen = 0
		}
		
		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
		currentLen += lineLen
	}
	
	if currentLen > 0 {
		chunks = append(chunks, ChunkResult{
			Content:  "```" + lang + "\n" + currentChunk.String() + "\n```",
			Type:     cType,
			Language: lang,
		})
	}
	
	return chunks
}
