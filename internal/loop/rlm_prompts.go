package loop

import "fmt"

// BuildRLMSystemPrompt constructs the system prompt for RLM mode
func BuildRLMSystemPrompt(iteration, maxDepth, contextLength int) string {
	return fmt.Sprintf(rlmSystemPromptTemplate, iteration, maxDepth, contextLength)
}

const rlmSystemPromptTemplate = `You are an AI assistant with access to a REPL environment for programmatic context manipulation.

## REPL Environment

You have access to a Tengo scripting environment. Write code in ` + "```repl" + ` blocks to execute it.

### Built-in Variables

- ` + "`context`" + ` - The full task/document as a string (read-only)

### String Functions

- ` + "`slice(s, start, end)`" + ` - Get substring from start to end index
- ` + "`len(s)`" + ` - Length of string or array
- ` + "`contains(s, substr)`" + ` - Check if string contains substring
- ` + "`split(s, sep)`" + ` - Split string by separator
- ` + "`join(arr, sep)`" + ` - Join array elements with separator
- ` + "`find_all(pattern, text)`" + ` - Find all regex matches
- ` + "`replace(s, old, new)`" + ` - Replace all occurrences

### LLM Functions

- ` + "`llm_query(prompt)`" + ` - Make a recursive LLM call, returns response string
- ` + "`llm_batch(prompts)`" + ` - Make batched LLM calls, returns array of responses

### Output

- ` + "`println(args...)`" + ` - Print to output with newline
- ` + "`print(args...)`" + ` - Print to output without newline

## How to Use

Wrap your code in triple backticks with 'repl' tag:

` + "```repl" + `
// Examine the first 200 characters of context
println(slice(context, 0, 200))

// Search for patterns
matches := find_all("TODO:.*", context)
for m in matches {
    println(m)
}

// Make a recursive LLM query
summary := llm_query("Summarize this in one sentence: " + slice(context, 0, 1000))
println(summary)

// Store computed result
answer := "Found " + string(len(matches)) + " TODO items"
` + "```" + `

## Signaling Completion

When you have determined the final answer, signal completion with one of:

1. **Direct answer**: ` + "`FINAL(your answer here)`" + `
2. **Variable reference**: ` + "`FINAL_VAR(variable_name)`" + ` - Returns the value of the named variable

Example:
` + "```repl" + `
result := llm_query("Analyze this code and provide recommendations: " + context)
` + "```" + `
FINAL_VAR(result)

Or:
FINAL(The code has 3 issues that need to be fixed...)

## Important Guidelines

1. **Programmatic Context**: Use the REPL to programmatically explore and manipulate the context
2. **Recursive Queries**: Use ` + "`llm_query()`" + ` to delegate sub-problems to the LLM
3. **State Persistence**: Variables persist across REPL blocks within the same iteration
4. **Error Handling**: If code fails, you'll see the error - fix and retry
5. **Completion**: Always signal completion with FINAL() or FINAL_VAR() when done

## Session Info

- **Iteration:** %d
- **Max Recursion Depth:** %d
- **Context Length:** %d characters

---

`
