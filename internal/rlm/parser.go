package rlm

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Patterns for FINAL("answer") extraction
	finalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`FINAL\s*\(\s*"""(.*)"""\s*\)`),      // FINAL("""answer""")
		regexp.MustCompile(`FINAL\s*\(\s*'''(.*)'''\s*\)`),      // FINAL('''answer''')
		regexp.MustCompile(`FINAL\s*\(\s*"([^"]*)"\s*\)`),       // FINAL("answer")
		regexp.MustCompile(`FINAL\s*\(\s*'([^']*)'\s*\)`),       // FINAL('answer')
		regexp.MustCompile("FINAL\\s*\\(\\s*`([^`]*)`\\s*\\)"),  // FINAL(`answer`)
	}

	// Pattern for FINAL_VAR(varname)
	finalVarPattern = regexp.MustCompile(`FINAL_VAR\s*\(\s*(\w+)\s*\)`)

	// Pattern for extracting code from markdown code blocks
	codeBlockPython = regexp.MustCompile("(?s)```python\\s*(.+?)```")
	codeBlockJS     = regexp.MustCompile("(?s)```javascript\\s*(.+?)```")
	codeBlockPlain  = regexp.MustCompile("(?s)```\\s*(.+?)```")
)

// IsFinal checks if the response contains a FINAL() or FINAL_VAR() statement.
func IsFinal(response string) bool {
	return strings.Contains(response, "FINAL(") || strings.Contains(response, "FINAL_VAR(")
}

// ExtractFinal extracts the answer from a FINAL("answer") statement.
// Returns the answer string and true if found, empty string and false otherwise.
func ExtractFinal(response string) (string, bool) {
	for _, pattern := range finalPatterns {
		// Use DOTALL mode via (?s) prefix in pattern for multi-line matching
		match := pattern.FindStringSubmatch(response)
		if len(match) >= 2 {
			return strings.TrimSpace(match[1]), true
		}
	}
	return "", false
}

// ExtractFinalVar extracts the variable name from FINAL_VAR(varname) and looks it up.
// Returns the variable value as string and true if found.
func ExtractFinalVar(response string, env *Environment) (string, bool) {
	match := finalVarPattern.FindStringSubmatch(response)
	if len(match) < 2 {
		return "", false
	}

	varName := match[1]

	// Look up in environment variables
	if value, ok := env.Variables[varName]; ok {
		return toString(value), true
	}

	return "", false
}

// ParseFinalStatement parses the response for any final statement.
// Returns the final answer and true if found.
func ParseFinalStatement(response string, env *Environment) (string, bool) {
	// Try FINAL() first (more common)
	if answer, ok := ExtractFinal(response); ok {
		return answer, true
	}

	// Try FINAL_VAR()
	if answer, ok := ExtractFinalVar(response, env); ok {
		return answer, true
	}

	return "", false
}

// ExtractCode extracts code from markdown code blocks if present.
// Returns the extracted code or the original text if no code blocks found.
func ExtractCode(text string) string {
	// Try Python code blocks first (preferred for RLM)
	if match := codeBlockPython.FindStringSubmatch(text); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}

	// Try JavaScript code blocks
	if match := codeBlockJS.FindStringSubmatch(text); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}

	// Try plain code blocks
	if match := codeBlockPlain.FindStringSubmatch(text); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}

	// No code blocks found, return original
	return text
}

// toString converts any value to a string representation.
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", val)
	}
}
