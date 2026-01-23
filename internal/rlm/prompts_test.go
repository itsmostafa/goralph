package rlm

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name          string
		contextSize   int
		depth         int
		query         string
		wantContains  []string
	}{
		{
			name:        "basic prompt",
			contextSize: 1000,
			depth:       0,
			query:       "What is this code about?",
			wantContains: []string{
				"Recursive Language Model",
				"JavaScript REPL",
				"context",
				"query",
				"What is this code about?",
				"FINAL",
				"Depth: 0",
			},
		},
		{
			name:        "large context with formatted number",
			contextSize: 1500000,
			depth:       2,
			query:       "Find the main function",
			wantContains: []string{
				"1,500,000 characters",
				"Depth: 2",
				"Find the main function",
			},
		},
		{
			name:        "includes fs module documentation",
			contextSize: 100,
			depth:       0,
			query:       "test",
			wantContains: []string{
				"fs.list",
				"fs.read",
				"fs.glob",
				"fs.exists",
				"fs.tree",
			},
		},
		{
			name:        "includes recursiveLLM documentation",
			contextSize: 100,
			depth:       0,
			query:       "test",
			wantContains: []string{
				"recursiveLLM",
				"subQuery",
				"subContext",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSystemPrompt(tt.contextSize, tt.depth, tt.query)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("BuildSystemPrompt() missing expected content %q", want)
				}
			}
		})
	}
}

func TestBuildCodePromptForQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "simple query",
			query: "What is the main function?",
			want:  "What is the main function?",
		},
		{
			name:  "empty query",
			query: "",
			want:  "",
		},
		{
			name:  "multiline query",
			query: "Find all\nerrors\nin the code",
			want:  "Find all\nerrors\nin the code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCodePromptForQuery(tt.query)
			if result != tt.want {
				t.Errorf("BuildCodePromptForQuery() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestBuildREPLResultPrompt(t *testing.T) {
	tests := []struct {
		name   string
		result *REPLResult
		want   string
	}{
		{
			name: "successful execution with output",
			result: &REPLResult{
				Output: "Hello, World!",
			},
			want: "Hello, World!",
		},
		{
			name: "successful execution no output",
			result: &REPLResult{
				Output: "",
			},
			want: "Code executed successfully (no output)",
		},
		{
			name: "error result",
			result: &REPLResult{
				Error: errTest,
			},
			want: "Error: test error",
		},
		{
			name: "truncated output",
			result: &REPLResult{
				Output:    "Some truncated content...",
				Truncated: true,
			},
			want: "Some truncated content...\n\n[Output truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildREPLResultPrompt(tt.result)
			if result != tt.want {
				t.Errorf("BuildREPLResultPrompt() = %q, want %q", result, tt.want)
			}
		})
	}
}

// Test error for use in tests
type testError struct{}

func (e testError) Error() string {
	return "test error"
}

var errTest = testError{}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{
			name: "zero",
			n:    0,
			want: "0",
		},
		{
			name: "small number",
			n:    42,
			want: "42",
		},
		{
			name: "hundreds",
			n:    999,
			want: "999",
		},
		{
			name: "one thousand",
			n:    1000,
			want: "1,000",
		},
		{
			name: "thousands",
			n:    12345,
			want: "12,345",
		},
		{
			name: "hundred thousands",
			n:    123456,
			want: "123,456",
		},
		{
			name: "millions",
			n:    1234567,
			want: "1,234,567",
		},
		{
			name: "billions",
			n:    1234567890,
			want: "1,234,567,890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNumber(tt.n)
			if result != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.n, result, tt.want)
			}
		})
	}
}
