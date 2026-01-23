package rlm

import (
	"context"
	"strings"
	"testing"
)

func TestREPLExecutor_BasicExecution(t *testing.T) {
	env := NewEnvironment("Hello, World!", "What is the context?")
	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), "context")

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "Hello, World!") {
		t.Errorf("expected output to contain context, got: %s", result.Output)
	}
}

func TestREPLExecutor_PrintFunction(t *testing.T) {
	env := NewEnvironment("test content", "test query")
	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), `print("Hello"); print("World")`)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "Hello") || !strings.Contains(result.Output, "World") {
		t.Errorf("expected output to contain printed values, got: %s", result.Output)
	}
}

func TestREPLExecutor_ContextSlice(t *testing.T) {
	env := NewEnvironment("0123456789ABCDEFGHIJ", "test")
	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), "context.slice(0, 10)")

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "0123456789") {
		t.Errorf("expected sliced context, got: %s", result.Output)
	}
}

func TestREPLExecutor_RegexFindAll(t *testing.T) {
	env := NewEnvironment("error: foo\nerror: bar\nwarning: baz", "find errors")
	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), `re.findAll("error: \\w+", context)`)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "error: foo") || !strings.Contains(result.Output, "error: bar") {
		t.Errorf("expected regex matches, got: %s", result.Output)
	}
}

func TestREPLExecutor_LineSplit(t *testing.T) {
	env := NewEnvironment("line1\nline2\nline3", "get lines")
	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), `context.split('\n').length`)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "3") {
		t.Errorf("expected 3 lines, got: %s", result.Output)
	}
}

func TestREPLExecutor_ErrorHandling(t *testing.T) {
	env := NewEnvironment("test", "test")
	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), "undefined_function()")

	if result.Error == nil {
		t.Error("expected an error for undefined function")
	}
}

func TestREPLExecutor_OutputTruncation(t *testing.T) {
	env := NewEnvironment(strings.Repeat("x", 5000), "test")
	config := DefaultConfig()
	config.MaxOutputChars = 100
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), "context")

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.Truncated {
		t.Error("expected output to be truncated")
	}
	if len(result.Output) > 100 {
		t.Errorf("expected output <= 100 chars, got: %d", len(result.Output))
	}
}

func TestREPLExecutor_QueryAccess(t *testing.T) {
	env := NewEnvironment("content", "What is the meaning of life?")
	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), "query")

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "What is the meaning of life?") {
		t.Errorf("expected query in output, got: %s", result.Output)
	}
}

func TestREPLExecutor_ConsoleLog(t *testing.T) {
	env := NewEnvironment("test", "test")
	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), `console.log("test output")`)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "test output") {
		t.Errorf("expected console.log output, got: %s", result.Output)
	}
}

func TestExtractVariableAssignments(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "let declaration",
			code:     "let x = 1",
			expected: []string{"x"},
		},
		{
			name:     "const declaration",
			code:     "const foo = 'bar'",
			expected: []string{"foo"},
		},
		{
			name:     "var declaration",
			code:     "var myVar = 123",
			expected: []string{"myVar"},
		},
		{
			name:     "multiple declarations",
			code:     "let a = 1\nconst b = 2\nvar c = 3",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "skip reserved words",
			code:     "let context = 1",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVariableAssignments(tt.code)
			if len(got) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, got)
				return
			}
			for i, name := range tt.expected {
				if got[i] != name {
					t.Errorf("expected %s at position %d, got %s", name, i, got[i])
				}
			}
		})
	}
}
