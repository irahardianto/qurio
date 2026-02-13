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

// CleanMarkdownNoise removes common documentation boilerplate from markdown
// before chunking. This is a pre-processing step that strips patterns that
// would never be useful in a code-assistance search context.
func CleanMarkdownNoise(text string) string {
	// Strip "Edit this page" style links
	editLinkRe := regexp.MustCompile(`(?mi)^\[edit[^\]]*\]\([^\)]+\)\s*$`)
	text = editLinkRe.ReplaceAllString(text, "")

	// Strip auto-generated "Table of Contents" sections
	// Match "## Table of Contents" or "## Contents" followed by link-only lines
	tocRe := regexp.MustCompile(`(?mi)^#{1,3}\s+(?:table of )?contents?\s*\n(?:\s*[-*]\s*\[.*?\]\(#.*?\)\s*\n)*`)
	text = tocRe.ReplaceAllString(text, "")

	return text
}

// IsNoiseChunk identifies chunks that are too low-value to embed.
// These are conservative heuristics — better to let a borderline chunk through
// than accidentally filter useful content.
func IsNoiseChunk(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return true
	}

	// Ultra-short labels (e.g., "Overview", "Getting Started") — no code, few words
	words := strings.Fields(trimmed)
	if len(trimmed) < 30 && len(words) <= 3 && !strings.Contains(trimmed, "```") && !strings.Contains(trimmed, "\n") {
		return true
	}

	// Install-only commands
	installRe := regexp.MustCompile(`(?mi)^\s*(npm|pnpm|yarn|pip|cargo|brew|apt|go)\s+(install|add|get|i)\b`)
	lines := strings.Split(trimmed, "\n")
	nonEmptyLines := filterNonEmpty(lines)
	if len(nonEmptyLines) > 0 && len(nonEmptyLines) <= 3 {
		allInstall := true
		for _, line := range nonEmptyLines {
			if !installRe.MatchString(line) {
				allInstall = false
				break
			}
		}
		if allInstall {
			return true
		}
	}

	// Pure navigation link lists (>70% of lines are markdown links)
	if len(nonEmptyLines) > 2 {
		linkRe := regexp.MustCompile(`^\s*[-*]?\s*\[.*?\]\(.*?\)\s*$`)
		linkCount := 0
		for _, line := range nonEmptyLines {
			if linkRe.MatchString(line) {
				linkCount++
			}
		}
		if float64(linkCount)/float64(len(nonEmptyLines)) > 0.7 {
			return true
		}
	}

	// Copyright/legal boilerplate
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "©") || strings.Contains(lower, "all rights reserved") ||
		strings.Contains(lower, "terms of service") || strings.Contains(lower, "privacy policy") {
		// Only noise if the chunk is short (not a full legal document that user intentionally indexed)
		if len(trimmed) < 200 {
			return true
		}
	}

	return false
}

func filterNonEmpty(lines []string) []string {
	var result []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			result = append(result, l)
		}
	}
	return result
}

// ChunkMarkdown implements a simplified chunker that splits text into chunks,
// preserving code blocks and identifying their language.
// It also splits large prose blocks into smaller chunks.
// Low-value noise chunks (install commands, nav links, etc.) are filtered out.
func ChunkMarkdown(text string, maxTokens, overlap int) []ChunkResult {
	// Pre-process: remove common documentation boilerplate
	text = CleanMarkdownNoise(text)

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
		} else if lang == "http" || lang == "graphql" || lang == "openapi" || lang == "swagger" {
			cType = ChunkTypeAPI
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

	// Post-filter: remove noise chunks
	filtered := make([]ChunkResult, 0, len(results))
	for _, chunk := range results {
		if !IsNoiseChunk(chunk.Content) {
			filtered = append(filtered, chunk)
		}
	}

	return filtered
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
			chunks = append(chunks, ChunkResult{Content: section, Type: detectChunkType(section)})
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
			if currentChunk.Len()+len(para)+2 <= maxChars {
				if currentChunk.Len() > 0 {
					currentChunk.WriteString("\n\n")
				}
				currentChunk.WriteString(para)
			} else {
				// Flush current chunk if not empty
				if currentChunk.Len() > 0 {
					chunks = append(chunks, ChunkResult{Content: currentChunk.String(), Type: detectChunkType(currentChunk.String())})
					currentChunk.Reset()
				}

				// Handle large paragraph
				if len(para) > maxChars {
					// 3. Split by Lines
					lines := strings.Split(para, "\n")
					for _, line := range lines {
						if currentChunk.Len()+len(line)+1 <= maxChars {
							if currentChunk.Len() > 0 {
								currentChunk.WriteString("\n")
							}
							currentChunk.WriteString(line)
						} else {
							if currentChunk.Len() > 0 {
								chunks = append(chunks, ChunkResult{Content: currentChunk.String(), Type: detectChunkType(currentChunk.String())})
								currentChunk.Reset()
							}

							// 4. Split by Words (Fallback)
							if len(line) > maxChars {
								words := strings.Fields(line)
								for _, word := range words {
									if currentChunk.Len()+len(word)+1 <= maxChars {
										if currentChunk.Len() > 0 {
											currentChunk.WriteString(" ")
										}
										currentChunk.WriteString(word)
									} else {
										chunks = append(chunks, ChunkResult{Content: currentChunk.String(), Type: detectChunkType(currentChunk.String())})
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
			chunks = append(chunks, ChunkResult{Content: currentChunk.String(), Type: detectChunkType(currentChunk.String())})
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

		if currentLen+lineLen > maxChars && currentLen > 0 {
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

func detectChunkType(content string) ChunkType {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "swagger") || strings.Contains(lower, "openapi") {
		return ChunkTypeAPI
	}
	// Heuristic: "Endpoint" and "Method" and "URL" usually means API doc
	if strings.Contains(lower, "endpoint") && strings.Contains(lower, "method") && (strings.Contains(lower, "url") || strings.Contains(lower, "http")) {
		return ChunkTypeAPI
	}
	return ChunkTypeProse
}
