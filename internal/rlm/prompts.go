package rlm

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt creates the system prompt for an RLM session.
// The prompt instructs the model on how to use the REPL environment.
func BuildSystemPrompt(contextSize int, depth int, query string) string {
	// Format context size with commas for readability
	contextSizeStr := formatNumber(contextSize)

	prompt := fmt.Sprintf(`You are a Recursive Language Model. You interact with context through a JavaScript REPL environment.

The context is stored in variable 'context' (not in this prompt). Size: %s characters.

Available in environment:
- context: string (the document/codebase to analyze)
- query: string (the question: "%s")
- recursiveLLM(subQuery, subContext): function that returns a string (recursively process sub-context)
- re: object with regex methods (findAll, search, split, replace)
- fs: object for filesystem exploration (list, read, glob, exists, tree)
- print(...args): function to output values

Filesystem functions (fs object):
- fs.list(path): List files in directory, returns [{name, isDir, size}]
- fs.read(path): Read file contents as string
- fs.glob(pattern): Find files matching glob pattern (e.g., "*.go", "**/*.ts")
- fs.exists(path): Check if path exists, returns boolean
- fs.tree(path, depth): Get directory tree up to depth, returns nested structure

Write JavaScript code to answer the query. The output of print() calls and the last expression value will be shown to you.

Examples:
- print(context.slice(0, 100))  // See first 100 chars
- context.split('\n').slice(0, 10)  // Get first 10 lines
- re.findAll('ERROR', context)  // Find all ERROR occurrences
- context.length  // Get context size
- fs.list('.')  // List files in current directory
- fs.read('main.go')  // Read a specific file
- fs.glob('**/*.go')  // Find all Go files
- let code = fs.read('src/main.ts')  // Load file into variable

When you have the answer, use FINAL("answer") - this is NOT a function call, just write it as text.
For answers built programmatically, use FINAL_VAR(varname) where varname holds the answer string.

IMPORTANT:
- Only write code when you need to explore the context
- When ready to answer, just write FINAL("your answer here")
- Keep code simple and focused on answering the query
- Use fs.* functions to explore the codebase and load specific files

Depth: %d`, contextSizeStr, query, depth)

	return prompt
}

// BuildCodePromptForQuery creates a prompt that includes the query for the user turn.
func BuildCodePromptForQuery(query string) string {
	return query
}

// BuildREPLResultPrompt formats REPL execution results for the conversation.
func BuildREPLResultPrompt(result *REPLResult) string {
	if result.Error != nil {
		return fmt.Sprintf("Error: %v", result.Error)
	}

	output := result.Output
	if output == "" {
		return "Code executed successfully (no output)"
	}

	if result.Truncated {
		return output + "\n\n[Output truncated]"
	}

	return output
}

// formatNumber formats an integer with comma separators.
func formatNumber(n int) string {
	str := fmt.Sprintf("%d", n)
	if n < 1000 {
		return str
	}

	var result strings.Builder
	length := len(str)

	for i, char := range str {
		if i > 0 && (length-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(char)
	}

	return result.String()
}
