package rlm

import (
	"testing"
)

func TestIsFinal(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{
			name:     "contains FINAL",
			response: `FINAL("The answer is 42")`,
			want:     true,
		},
		{
			name:     "contains FINAL_VAR",
			response: `FINAL_VAR(answer)`,
			want:     true,
		},
		{
			name:     "no final statement",
			response: "Let me explore the context more",
			want:     false,
		},
		{
			name:     "final in text but not statement",
			response: "The final result is unclear",
			want:     false,
		},
		{
			name:     "empty response",
			response: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFinal(tt.response)
			if result != tt.want {
				t.Errorf("IsFinal() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestExtractFinal(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
		wantOk   bool
	}{
		{
			name:     "double quoted",
			response: `FINAL("The answer is 42")`,
			want:     "The answer is 42",
			wantOk:   true,
		},
		{
			name:     "single quoted",
			response: `FINAL('The answer is 42')`,
			want:     "The answer is 42",
			wantOk:   true,
		},
		{
			name:     "backtick quoted",
			response: "FINAL(`The answer is 42`)",
			want:     "The answer is 42",
			wantOk:   true,
		},
		{
			name:     "triple double quoted",
			response: `FINAL("""The answer is 42""")`,
			want:     "The answer is 42",
			wantOk:   true,
		},
		{
			name:     "triple single quoted",
			response: `FINAL('''The answer is 42''')`,
			want:     "The answer is 42",
			wantOk:   true,
		},
		{
			name:     "with whitespace",
			response: `FINAL(  "The answer is 42"  )`,
			want:     "The answer is 42",
			wantOk:   true,
		},
		{
			name:     "answer with leading/trailing spaces",
			response: `FINAL("  spaced answer  ")`,
			want:     "spaced answer",
			wantOk:   true,
		},
		{
			name:     "no final statement",
			response: "Just some regular text",
			want:     "",
			wantOk:   false,
		},
		{
			name:     "empty response",
			response: "",
			want:     "",
			wantOk:   false,
		},
		{
			name:     "embedded in text",
			response: "After analysis, I found FINAL(\"The answer\") to be correct.",
			want:     "The answer",
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractFinal(tt.response)
			if ok != tt.wantOk {
				t.Errorf("ExtractFinal() ok = %v, want %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ExtractFinal() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFinalVar(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		variables map[string]any
		want      string
		wantOk    bool
	}{
		{
			name:      "variable exists as string",
			response:  "FINAL_VAR(answer)",
			variables: map[string]any{"answer": "The answer is 42"},
			want:      "The answer is 42",
			wantOk:    true,
		},
		{
			name:      "variable exists as int",
			response:  "FINAL_VAR(result)",
			variables: map[string]any{"result": 42},
			want:      "42",
			wantOk:    true,
		},
		{
			name:      "variable exists as nil",
			response:  "FINAL_VAR(empty)",
			variables: map[string]any{"empty": nil},
			want:      "",
			wantOk:    true,
		},
		{
			name:      "variable does not exist",
			response:  "FINAL_VAR(missing)",
			variables: map[string]any{"other": "value"},
			want:      "",
			wantOk:    false,
		},
		{
			name:      "with whitespace",
			response:  "FINAL_VAR(  answer  )",
			variables: map[string]any{"answer": "test"},
			want:      "test",
			wantOk:    true,
		},
		{
			name:      "no FINAL_VAR statement",
			response:  "Just some text",
			variables: map[string]any{"answer": "test"},
			want:      "",
			wantOk:    false,
		},
		{
			name:      "empty variables map",
			response:  "FINAL_VAR(answer)",
			variables: map[string]any{},
			want:      "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &Environment{Variables: tt.variables}
			got, ok := ExtractFinalVar(tt.response, env)
			if ok != tt.wantOk {
				t.Errorf("ExtractFinalVar() ok = %v, want %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ExtractFinalVar() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseFinalStatement(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		variables map[string]any
		want      string
		wantOk    bool
	}{
		{
			name:      "FINAL takes precedence",
			response:  `FINAL("direct answer")`,
			variables: map[string]any{},
			want:      "direct answer",
			wantOk:    true,
		},
		{
			name:      "FINAL_VAR when no FINAL",
			response:  "FINAL_VAR(myvar)",
			variables: map[string]any{"myvar": "variable answer"},
			want:      "variable answer",
			wantOk:    true,
		},
		{
			name:      "no final statement",
			response:  "Still exploring...",
			variables: map[string]any{"myvar": "value"},
			want:      "",
			wantOk:    false,
		},
		{
			name:      "FINAL_VAR with missing variable",
			response:  "FINAL_VAR(missing)",
			variables: map[string]any{"other": "value"},
			want:      "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &Environment{Variables: tt.variables}
			got, ok := ParseFinalStatement(tt.response, env)
			if ok != tt.wantOk {
				t.Errorf("ParseFinalStatement() ok = %v, want %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("ParseFinalStatement() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractCode(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "python code block",
			text: "Here's the code:\n```python\nprint('hello')\n```",
			want: "print('hello')",
		},
		{
			name: "javascript code block",
			text: "```javascript\nconsole.log('hello');\n```",
			want: "console.log('hello');",
		},
		{
			name: "plain code block",
			text: "```\nlet x = 1;\n```",
			want: "let x = 1;",
		},
		{
			name: "no code block",
			text: "Just some text",
			want: "Just some text",
		},
		{
			name: "python preferred over plain",
			text: "```python\npython code\n```\nand\n```\nplain code\n```",
			want: "python code",
		},
		{
			name: "multiline code",
			text: "```javascript\nlet a = 1;\nlet b = 2;\nprint(a + b);\n```",
			want: "let a = 1;\nlet b = 2;\nprint(a + b);",
		},
		{
			name: "code with surrounding text",
			text: "Let me analyze:\n```python\ncontext.slice(0,100)\n```\nThis shows...",
			want: "context.slice(0,100)",
		},
		{
			name: "empty code block",
			text: "```python\n```",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCode(tt.text)
			if got != tt.want {
				t.Errorf("ExtractCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "string value",
			value: "hello",
			want:  "hello",
		},
		{
			name:  "byte slice",
			value: []byte("world"),
			want:  "world",
		},
		{
			name:  "nil value",
			value: nil,
			want:  "",
		},
		{
			name:  "int value",
			value: 42,
			want:  "42",
		},
		{
			name:  "float value",
			value: 3.14,
			want:  "3.14",
		},
		{
			name:  "bool value",
			value: true,
			want:  "true",
		},
		{
			name:  "slice value",
			value: []int{1, 2, 3},
			want:  "[1 2 3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toString(tt.value)
			if got != tt.want {
				t.Errorf("toString() = %q, want %q", got, tt.want)
			}
		})
	}
}
