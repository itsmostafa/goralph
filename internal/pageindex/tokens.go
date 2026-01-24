package pageindex

import (
	"strings"
	"unicode"
)

// CountTokens provides a simple token count approximation.
// For accurate counting, use a proper tokenizer like tiktoken.
// This approximation uses ~4 characters per token on average.
func CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Count words and punctuation as rough token estimate
	// Most tokenizers produce ~1.3 tokens per word on average
	words := strings.Fields(text)
	wordCount := len(words)

	// Add extra for punctuation (typically separate tokens)
	punctCount := 0
	for _, r := range text {
		if unicode.IsPunct(r) {
			punctCount++
		}
	}

	// Approximate: words * 1.3 + punctuation
	return int(float64(wordCount)*1.3) + punctCount/2
}

// CountTokensForModel returns a token counter function for a specific model.
// Currently returns the simple approximation; can be extended with tiktoken.
func CountTokensForModel(model string) func(string) int {
	// TODO: Integrate with tiktoken for accurate counting
	// For now, use approximation
	return CountTokens
}

// EstimateTokensPerPage estimates tokens for a page of PDF text.
// Average is about 500-800 tokens per page depending on content.
func EstimateTokensPerPage(pageText string) int {
	return CountTokens(pageText)
}

// SplitByTokens splits text into chunks of approximately maxTokens each.
func SplitByTokens(text string, maxTokens int) []string {
	if maxTokens <= 0 {
		return []string{text}
	}

	tokens := CountTokens(text)
	if tokens <= maxTokens {
		return []string{text}
	}

	// Split by paragraphs first, then by sentences if needed
	paragraphs := strings.Split(text, "\n\n")

	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, para := range paragraphs {
		paraTokens := CountTokens(para)

		if currentTokens+paraTokens > maxTokens && currentChunk.Len() > 0 {
			// Save current chunk and start new one
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
			currentTokens = 0
		}

		if paraTokens > maxTokens {
			// Paragraph too large, split by sentences
			sentences := splitIntoSentences(para)
			for _, sent := range sentences {
				sentTokens := CountTokens(sent)
				if currentTokens+sentTokens > maxTokens && currentChunk.Len() > 0 {
					chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
					currentChunk.Reset()
					currentTokens = 0
				}
				if currentChunk.Len() > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(sent)
				currentTokens += sentTokens
			}
		} else {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
			currentTokens += paraTokens
		}
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}

// splitIntoSentences splits text into sentences.
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	// Handle remaining text
	if current.Len() > 0 {
		s := strings.TrimSpace(current.String())
		if s != "" {
			sentences = append(sentences, s)
		}
	}

	return sentences
}

// GroupPagesByTokens groups pages into chunks that fit within maxTokens.
// Returns slices of page content with overlap for context continuity.
func GroupPagesByTokens(pages []PageContent, maxTokens, overlapPages int) []string {
	if len(pages) == 0 {
		return nil
	}

	totalTokens := 0
	for _, p := range pages {
		totalTokens += p.TokenCount
	}

	if totalTokens <= maxTokens {
		// All fits in one group
		var sb strings.Builder
		for _, p := range pages {
			sb.WriteString(p.Text)
		}
		return []string{sb.String()}
	}

	// Calculate expected parts for even distribution
	expectedParts := (totalTokens + maxTokens - 1) / maxTokens
	avgTokensPerPart := (totalTokens/expectedParts + maxTokens) / 2

	var groups []string
	var current strings.Builder
	currentTokens := 0
	startIdx := 0

	for i, page := range pages {
		if currentTokens+page.TokenCount > avgTokensPerPart && current.Len() > 0 {
			// Save current group
			groups = append(groups, current.String())

			// Start new group with overlap
			current.Reset()
			currentTokens = 0
			overlapStart := i - overlapPages
			if overlapStart < startIdx {
				overlapStart = startIdx
			}
			for j := overlapStart; j < i; j++ {
				current.WriteString(pages[j].Text)
				currentTokens += pages[j].TokenCount
			}
			startIdx = i
		}

		current.WriteString(page.Text)
		currentTokens += page.TokenCount
	}

	if current.Len() > 0 {
		groups = append(groups, current.String())
	}

	return groups
}
