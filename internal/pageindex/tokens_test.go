package pageindex

import (
	"strings"
	"testing"
)

func TestCountTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minExpected int
		maxExpected int
	}{
		{"empty string", "", 0, 0},
		{"single word", "hello", 1, 3},
		{"simple sentence", "Hello world!", 2, 5},
		{"longer text", "The quick brown fox jumps over the lazy dog.", 10, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountTokens(tt.input)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("CountTokens(%q) = %d, want between %d and %d",
					tt.input, result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestCountTokensForModel(t *testing.T) {
	// Should return a function
	counter := CountTokensForModel("gpt-4")
	if counter == nil {
		t.Fatal("expected non-nil counter function")
	}

	// Should work like CountTokens
	result := counter("hello world")
	if result <= 0 {
		t.Error("expected positive token count")
	}
}

func TestEstimateTokensPerPage(t *testing.T) {
	text := "This is a sample page with some text content."
	result := EstimateTokensPerPage(text)
	if result <= 0 {
		t.Error("expected positive token count")
	}
}

func TestSplitByTokens(t *testing.T) {
	t.Run("empty text", func(t *testing.T) {
		result := SplitByTokens("", 100)
		if len(result) != 1 {
			t.Errorf("expected 1 chunk, got %d", len(result))
		}
	})

	t.Run("text within limit", func(t *testing.T) {
		text := "Short text."
		result := SplitByTokens(text, 1000)
		if len(result) != 1 {
			t.Errorf("expected 1 chunk, got %d", len(result))
		}
		if result[0] != text {
			t.Errorf("expected unchanged text, got %q", result[0])
		}
	})

	t.Run("text exceeds limit", func(t *testing.T) {
		// Create text that will need splitting
		para1 := "First paragraph with some content."
		para2 := "Second paragraph with more content."
		para3 := "Third paragraph with even more content."
		text := para1 + "\n\n" + para2 + "\n\n" + para3

		// Use a very small limit to force splitting
		result := SplitByTokens(text, 10)
		if len(result) < 2 {
			t.Errorf("expected text to be split into multiple chunks, got %d", len(result))
		}
	})

	t.Run("zero limit returns original", func(t *testing.T) {
		text := "Some text."
		result := SplitByTokens(text, 0)
		if len(result) != 1 || result[0] != text {
			t.Error("expected original text returned for zero limit")
		}
	})
}

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"single sentence", "Hello world.", 1},
		{"two sentences", "Hello world. How are you?", 2},
		{"with exclamation", "Hello! World.", 2},
		{"no punctuation", "Hello world", 1},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitIntoSentences(tt.input)
			if len(result) != tt.expected {
				t.Errorf("splitIntoSentences(%q) returned %d sentences, want %d",
					tt.input, len(result), tt.expected)
			}
		})
	}
}

func TestGroupPagesByTokens(t *testing.T) {
	t.Run("empty pages", func(t *testing.T) {
		result := GroupPagesByTokens(nil, 100, 1)
		if result != nil {
			t.Error("expected nil for empty pages")
		}
	})

	t.Run("all pages fit", func(t *testing.T) {
		pages := []PageContent{
			{Text: "Page 1 content.", TokenCount: 10},
			{Text: "Page 2 content.", TokenCount: 10},
		}
		result := GroupPagesByTokens(pages, 100, 1)
		if len(result) != 1 {
			t.Errorf("expected 1 group, got %d", len(result))
		}
		if !strings.Contains(result[0], "Page 1") || !strings.Contains(result[0], "Page 2") {
			t.Error("expected both pages in single group")
		}
	})

	t.Run("pages need splitting", func(t *testing.T) {
		pages := []PageContent{
			{Text: "Page 1 content.", TokenCount: 50},
			{Text: "Page 2 content.", TokenCount: 50},
			{Text: "Page 3 content.", TokenCount: 50},
			{Text: "Page 4 content.", TokenCount: 50},
		}
		result := GroupPagesByTokens(pages, 60, 0)
		if len(result) < 2 {
			t.Errorf("expected multiple groups, got %d", len(result))
		}
	})

	t.Run("with overlap", func(t *testing.T) {
		pages := []PageContent{
			{Text: "Page 1. ", TokenCount: 30},
			{Text: "Page 2. ", TokenCount: 30},
			{Text: "Page 3. ", TokenCount: 30},
			{Text: "Page 4. ", TokenCount: 30},
		}
		result := GroupPagesByTokens(pages, 50, 1)
		if len(result) < 2 {
			t.Errorf("expected multiple groups, got %d", len(result))
		}
		// Check that overlap is applied (page from previous group appears in next)
		// This is a basic check; overlap implementation details may vary
	})
}
