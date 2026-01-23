package rlm

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// REPLExecutor executes JavaScript code in a sandboxed goja environment.
type REPLExecutor struct {
	env    *Environment
	config Config
}

// NewREPLExecutor creates a new REPL executor with the given environment and config.
func NewREPLExecutor(env *Environment, config Config) *REPLExecutor {
	return &REPLExecutor{
		env:    env,
		config: config,
	}
}

// Execute runs JavaScript code in the sandboxed environment.
// It returns the output from print() calls and the final expression value.
func (r *REPLExecutor) Execute(ctx context.Context, code string) *REPLResult {
	// Create a new goja runtime for each execution (isolation)
	vm := goja.New()

	// Set up interrupt handling for timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(r.config.Timeout)*time.Second)
	defer cancel()

	// Set up interrupt for context cancellation
	go func() {
		<-timeoutCtx.Done()
		vm.Interrupt("execution timeout or cancelled")
	}()

	// Collect printed output
	var printedOutput strings.Builder

	// Set up the environment in the VM
	if err := r.setupEnvironment(vm, &printedOutput); err != nil {
		return &REPLResult{Error: fmt.Errorf("failed to setup environment: %w", err)}
	}

	// Execute the code
	val, err := vm.RunString(code)
	if err != nil {
		// Check if it was an interrupt
		if interrupted, ok := err.(*goja.InterruptedError); ok {
			return &REPLResult{Error: fmt.Errorf("execution interrupted: %s", interrupted.Value())}
		}
		return &REPLResult{Error: fmt.Errorf("execution error: %w", err)}
	}

	// Build the output
	output := r.buildOutput(&printedOutput, val)

	// Check if truncation is needed
	truncated := false
	if len(output) > r.config.MaxOutputChars {
		output = output[:r.config.MaxOutputChars]
		truncated = true
	}

	return &REPLResult{
		Output:    output,
		Truncated: truncated,
	}
}

// setupEnvironment configures the goja VM with the RLM environment.
func (r *REPLExecutor) setupEnvironment(vm *goja.Runtime, printOutput *strings.Builder) error {
	// Set the context variable
	if err := vm.Set("context", r.env.Context); err != nil {
		return fmt.Errorf("failed to set context: %w", err)
	}

	// Set the query variable
	if err := vm.Set("query", r.env.Query); err != nil {
		return fmt.Errorf("failed to set query: %w", err)
	}

	// Set up the print function
	printFunc := func(call goja.FunctionCall) goja.Value {
		args := make([]string, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.String()
		}
		printOutput.WriteString(strings.Join(args, " "))
		printOutput.WriteString("\n")
		return goja.Undefined()
	}
	if err := vm.Set("print", printFunc); err != nil {
		return fmt.Errorf("failed to set print: %w", err)
	}

	// Set up the console.log as an alias for print
	console := vm.NewObject()
	if err := console.Set("log", printFunc); err != nil {
		return fmt.Errorf("failed to set console.log: %w", err)
	}
	if err := vm.Set("console", console); err != nil {
		return fmt.Errorf("failed to set console: %w", err)
	}

	// Set up the regex module
	if err := r.setupRegexModule(vm); err != nil {
		return fmt.Errorf("failed to setup regex module: %w", err)
	}

	// Set up user variables from previous executions
	for name, value := range r.env.Variables {
		if err := vm.Set(name, value); err != nil {
			return fmt.Errorf("failed to set variable %s: %w", name, err)
		}
	}

	// Set up recursiveLLM function (if available)
	if r.env.RecursiveLLM != nil && r.env.CurrentDepth < r.env.MaxDepth {
		recursiveFunc := func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) < 2 {
				panic(vm.NewTypeError("recursiveLLM requires 2 arguments: subQuery, subContext"))
			}
			subQuery := call.Arguments[0].String()
			subContext := call.Arguments[1].String()

			result, err := r.env.RecursiveLLM(subQuery, subContext)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			return vm.ToValue(result)
		}
		if err := vm.Set("recursiveLLM", recursiveFunc); err != nil {
			return fmt.Errorf("failed to set recursiveLLM: %w", err)
		}
	} else {
		// Set a placeholder that returns an error message
		placeholderFunc := func(call goja.FunctionCall) goja.Value {
			if r.env.CurrentDepth >= r.env.MaxDepth {
				return vm.ToValue("Error: Maximum recursion depth reached")
			}
			return vm.ToValue("Error: Recursive LLM not available")
		}
		if err := vm.Set("recursiveLLM", placeholderFunc); err != nil {
			return fmt.Errorf("failed to set recursiveLLM placeholder: %w", err)
		}
	}

	return nil
}

// setupRegexModule adds the 're' object with regex helper functions.
func (r *REPLExecutor) setupRegexModule(vm *goja.Runtime) error {
	re := vm.NewObject()

	// re.findAll(pattern, text) -> array of matches
	findAll := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("findAll requires 2 arguments: pattern, text"))
		}
		pattern := call.Arguments[0].String()
		text := call.Arguments[1].String()

		matches, err := r.env.RE.FindAll(pattern, text)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(matches)
	}
	if err := re.Set("findAll", findAll); err != nil {
		return err
	}

	// re.search(pattern, text) -> first match or empty string
	search := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("search requires 2 arguments: pattern, text"))
		}
		pattern := call.Arguments[0].String()
		text := call.Arguments[1].String()

		match, err := r.env.RE.Search(pattern, text)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(match)
	}
	if err := re.Set("search", search); err != nil {
		return err
	}

	// re.split(pattern, text, n) -> array of strings
	split := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("split requires at least 2 arguments: pattern, text"))
		}
		pattern := call.Arguments[0].String()
		text := call.Arguments[1].String()
		n := -1 // unlimited by default
		if len(call.Arguments) >= 3 {
			n = int(call.Arguments[2].ToInteger())
		}

		parts, err := r.env.RE.Split(pattern, text, n)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(parts)
	}
	if err := re.Set("split", split); err != nil {
		return err
	}

	// re.replace(pattern, text, replacement) -> replaced string
	replace := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.NewTypeError("replace requires 3 arguments: pattern, text, replacement"))
		}
		pattern := call.Arguments[0].String()
		text := call.Arguments[1].String()
		replacement := call.Arguments[2].String()

		result, err := r.env.RE.Replace(pattern, text, replacement)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(result)
	}
	if err := re.Set("replace", replace); err != nil {
		return err
	}

	return vm.Set("re", re)
}

// buildOutput constructs the final output from print statements and expression result.
func (r *REPLExecutor) buildOutput(printOutput *strings.Builder, val goja.Value) string {
	var output strings.Builder

	// Include printed output
	printed := printOutput.String()
	if printed != "" {
		output.WriteString(printed)
	}

	// Include the final expression value if it's not undefined
	if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
		// Format the value for display
		valStr := r.formatValue(val)
		if valStr != "" {
			if output.Len() > 0 && !strings.HasSuffix(output.String(), "\n") {
				output.WriteString("\n")
			}
			output.WriteString("=> ")
			output.WriteString(valStr)
		}
	}

	return output.String()
}

// formatValue formats a goja value for display in REPL output.
func (r *REPLExecutor) formatValue(val goja.Value) string {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return ""
	}

	// Try to export to Go value for better formatting
	exported := val.Export()

	switch v := exported.(type) {
	case string:
		// Truncate very long strings
		if len(v) > 1000 {
			return fmt.Sprintf("%q... (truncated, total %d chars)", v[:1000], len(v))
		}
		return fmt.Sprintf("%q", v)
	case []any:
		// Format arrays nicely
		if len(v) == 0 {
			return "[]"
		}
		if len(v) > 20 {
			// Truncate large arrays
			items := make([]string, 21)
			for i := range 20 {
				items[i] = fmt.Sprintf("%v", v[i])
			}
			items[20] = fmt.Sprintf("... (%d more items)", len(v)-20)
			return "[" + strings.Join(items, ", ") + "]"
		}
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = fmt.Sprintf("%v", item)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case map[string]any:
		// Format objects
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// UpdateEnvironment updates the environment with variables from the last execution.
// This should be called after Execute to persist any variables defined in the code.
func (r *REPLExecutor) UpdateEnvironment(vm *goja.Runtime, varNames []string) {
	for _, name := range varNames {
		val := vm.Get(name)
		if val != nil && !goja.IsUndefined(val) {
			r.env.Variables[name] = val.Export()
		}
	}
}

// ExtractVariableAssignments parses code to find variable assignments.
// Returns a list of variable names that were assigned.
func ExtractVariableAssignments(code string) []string {
	// Simple regex-based extraction for common patterns:
	// let x = ..., const x = ..., var x = ..., x = ...
	patterns := []string{
		`\b(?:let|const|var)\s+(\w+)\s*=`,
		`^(\w+)\s*=`,
	}

	seen := make(map[string]bool)
	var names []string

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				name := match[1]
				// Skip reserved words
				if !isReservedWord(name) && !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
		}
	}

	return names
}

// isReservedWord checks if a name is a JavaScript reserved word or built-in.
func isReservedWord(name string) bool {
	reserved := map[string]bool{
		"break": true, "case": true, "catch": true, "continue": true,
		"debugger": true, "default": true, "delete": true, "do": true,
		"else": true, "finally": true, "for": true, "function": true,
		"if": true, "in": true, "instanceof": true, "new": true,
		"return": true, "switch": true, "this": true, "throw": true,
		"try": true, "typeof": true, "var": true, "void": true,
		"while": true, "with": true, "let": true, "const": true,
		"class": true, "export": true, "extends": true, "import": true,
		"super": true, "yield": true, "true": true, "false": true,
		"null": true, "undefined": true,
		// Built-ins from our environment
		"context": true, "query": true, "print": true, "console": true,
		"re": true, "recursiveLLM": true,
	}
	return reserved[name]
}

