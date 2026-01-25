package loop

import (
	"regexp"
	"strings"
)

// REPL code block pattern: ```repl ... ```
// The closing ``` must be at the start of a line (after newline) to avoid
// matching backticks inside the code (which are raw string literals in Tengo)
var replBlockPattern = regexp.MustCompile("(?s)```repl[^\\n]*\\n(.*?)\\n```")

// FINAL marker patterns
var finalAnswerPattern = regexp.MustCompile(`FINAL\(([^)]+)\)`)
var finalVarPattern = regexp.MustCompile(`FINAL_VAR\(([^)]+)\)`)

// ExtractREPLCodeBlocks extracts all code blocks marked with ```repl from LLM output
func ExtractREPLCodeBlocks(text string) []string {
	matches := replBlockPattern.FindAllStringSubmatch(text, -1)
	blocks := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 2 {
			code := strings.TrimSpace(match[1])
			if code != "" {
				blocks = append(blocks, code)
			}
		}
	}
	return blocks
}

// ExtractFinalAnswer extracts the answer from FINAL(answer) marker
// Returns empty string if not found
func ExtractFinalAnswer(text string) string {
	match := finalAnswerPattern.FindStringSubmatch(text)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// ExtractFinalVar extracts the variable name from FINAL_VAR(variable_name) marker
// Returns empty string if not found
func ExtractFinalVar(text string) string {
	match := finalVarPattern.FindStringSubmatch(text)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// HasCompletionMarker checks if the text contains any completion marker
func HasCompletionMarker(text string) bool {
	return finalAnswerPattern.MatchString(text) ||
		finalVarPattern.MatchString(text) ||
		strings.Contains(text, CompletionPromise)
}
