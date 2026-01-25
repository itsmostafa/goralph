package loop

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/d5/tengo/v2"
)

// GoREPL implements REPLExecutor using the Tengo scripting language
type GoREPL struct {
	bridge    LMBridge
	variables map[string]any
	output    *strings.Builder
	llmCalls  []LLMCallRecord
	ctx       string // The context variable
}

// NewGoREPL creates a new Go-based REPL with the given LM bridge
func NewGoREPL(bridge LMBridge) *GoREPL {
	return &GoREPL{
		bridge:    bridge,
		variables: make(map[string]any),
		output:    &strings.Builder{},
	}
}

// Initialize sets up the REPL environment
func (r *GoREPL) Initialize() error {
	r.variables = make(map[string]any)
	r.output.Reset()
	r.llmCalls = nil
	return nil
}

// SetContext loads context into the REPL as the "context" variable
func (r *GoREPL) SetContext(ctx string) error {
	r.ctx = ctx
	r.variables["context"] = ctx
	return nil
}

// Execute runs Tengo code and returns the result
func (r *GoREPL) Execute(code string) (*REPLResult, error) {
	startTime := time.Now()
	r.output.Reset()
	r.llmCalls = nil

	// Create a new script
	script := tengo.NewScript([]byte(code))

	// Set resource limits for sandboxing
	script.SetMaxAllocs(1000000) // Limit memory allocations

	// Add the context variable
	if err := script.Add("context", r.ctx); err != nil {
		return &REPLResult{Error: fmt.Sprintf("failed to add context: %v", err)}, nil
	}

	// Add persisted variables
	for name, value := range r.variables {
		if name == "context" {
			continue // Already added
		}
		tengoVal := r.toTengoValue(value)
		if err := script.Add(name, tengoVal); err != nil {
			// Ignore errors for complex types
			continue
		}
	}

	// Add built-in functions
	r.addBuiltinFunctions(script)

	// Compile the script
	compiled, err := script.Compile()
	if err != nil {
		return &REPLResult{Error: fmt.Sprintf("compile error: %v", err)}, nil
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = compiled.RunContext(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &REPLResult{Error: "execution timed out after 30s"}, nil
		}
		return &REPLResult{Error: fmt.Sprintf("runtime error: %v", err)}, nil
	}

	// Extract variables from compiled script for state persistence
	r.extractVariables(compiled)

	return &REPLResult{
		Output:     r.output.String(),
		Variables:  r.variables,
		ExecTimeMs: int(time.Since(startTime).Milliseconds()),
		LLMCalls:   r.llmCalls,
	}, nil
}

// addBuiltinFunctions adds custom functions to the script
func (r *GoREPL) addBuiltinFunctions(script *tengo.Script) {
	// println function
	_ = script.Add("println", &tengo.UserFunction{
		Name: "println",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			for i, arg := range args {
				if i > 0 {
					r.output.WriteString(" ")
				}
				r.output.WriteString(objectToString(arg))
			}
			r.output.WriteString("\n")
			return tengo.UndefinedValue, nil
		},
	})

	// print function (no newline)
	_ = script.Add("print", &tengo.UserFunction{
		Name: "print",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			for i, arg := range args {
				if i > 0 {
					r.output.WriteString(" ")
				}
				r.output.WriteString(objectToString(arg))
			}
			return tengo.UndefinedValue, nil
		},
	})

	// slice function
	_ = script.Add("slice", &tengo.UserFunction{
		Name: "slice",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 3 {
				return nil, tengo.ErrWrongNumArguments
			}
			s, ok := tengo.ToString(args[0])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			start, ok := tengo.ToInt(args[1])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "second", Expected: "int", Found: args[1].TypeName()}
			}
			end, ok := tengo.ToInt(args[2])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "third", Expected: "int", Found: args[2].TypeName()}
			}
			// Bounds checking
			if start < 0 {
				start = 0
			}
			if end > len(s) {
				end = len(s)
			}
			if start > end {
				return &tengo.String{Value: ""}, nil
			}
			return &tengo.String{Value: s[start:end]}, nil
		},
	})

	// len function (override default to work with our context)
	_ = script.Add("len", &tengo.UserFunction{
		Name: "len",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			switch v := args[0].(type) {
			case *tengo.String:
				return &tengo.Int{Value: int64(len(v.Value))}, nil
			case *tengo.Array:
				return &tengo.Int{Value: int64(len(v.Value))}, nil
			case *tengo.Map:
				return &tengo.Int{Value: int64(len(v.Value))}, nil
			default:
				return &tengo.Int{Value: 0}, nil
			}
		},
	})

	// contains function
	_ = script.Add("contains", &tengo.UserFunction{
		Name: "contains",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return nil, tengo.ErrWrongNumArguments
			}
			s, ok := tengo.ToString(args[0])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			substr, ok := tengo.ToString(args[1])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "second", Expected: "string", Found: args[1].TypeName()}
			}
			if strings.Contains(s, substr) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	})

	// split function
	_ = script.Add("split", &tengo.UserFunction{
		Name: "split",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return nil, tengo.ErrWrongNumArguments
			}
			s, ok := tengo.ToString(args[0])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			sep, ok := tengo.ToString(args[1])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "second", Expected: "string", Found: args[1].TypeName()}
			}
			parts := strings.Split(s, sep)
			arr := make([]tengo.Object, len(parts))
			for i, p := range parts {
				arr[i] = &tengo.String{Value: p}
			}
			return &tengo.Array{Value: arr}, nil
		},
	})

	// join function
	_ = script.Add("join", &tengo.UserFunction{
		Name: "join",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return nil, tengo.ErrWrongNumArguments
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "array", Found: args[0].TypeName()}
			}
			sep, ok := tengo.ToString(args[1])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "second", Expected: "string", Found: args[1].TypeName()}
			}
			strs := make([]string, len(arr.Value))
			for i, v := range arr.Value {
				strs[i] = objectToString(v)
			}
			return &tengo.String{Value: strings.Join(strs, sep)}, nil
		},
	})

	// find_all function (regex)
	_ = script.Add("find_all", &tengo.UserFunction{
		Name: "find_all",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return nil, tengo.ErrWrongNumArguments
			}
			pattern, ok := tengo.ToString(args[0])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			text, ok := tengo.ToString(args[1])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "second", Expected: "string", Found: args[1].TypeName()}
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex: %v", err)
			}
			matches := re.FindAllString(text, -1)
			arr := make([]tengo.Object, len(matches))
			for i, m := range matches {
				arr[i] = &tengo.String{Value: m}
			}
			return &tengo.Array{Value: arr}, nil
		},
	})

	// replace function
	_ = script.Add("replace", &tengo.UserFunction{
		Name: "replace",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 3 {
				return nil, tengo.ErrWrongNumArguments
			}
			s, ok := tengo.ToString(args[0])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			old, ok := tengo.ToString(args[1])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "second", Expected: "string", Found: args[1].TypeName()}
			}
			newStr, ok := tengo.ToString(args[2])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "third", Expected: "string", Found: args[2].TypeName()}
			}
			return &tengo.String{Value: strings.ReplaceAll(s, old, newStr)}, nil
		},
	})

	// llm_query function
	_ = script.Add("llm_query", &tengo.UserFunction{
		Name: "llm_query",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			prompt, ok := tengo.ToString(args[0])
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			if r.bridge == nil {
				return nil, fmt.Errorf("LLM bridge not configured")
			}
			response, err := r.bridge.HandleRequest(prompt)
			if err != nil {
				return &tengo.Error{Value: &tengo.String{Value: err.Error()}}, nil
			}
			// Track the call
			r.llmCalls = append(r.llmCalls, LLMCallRecord{Prompt: prompt, Response: response})
			return &tengo.String{Value: response}, nil
		},
	})

	// llm_batch function
	_ = script.Add("llm_batch", &tengo.UserFunction{
		Name: "llm_batch",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "array", Found: args[0].TypeName()}
			}
			if r.bridge == nil {
				return nil, fmt.Errorf("LLM bridge not configured")
			}
			prompts := make([]string, len(arr.Value))
			for i, v := range arr.Value {
				s, ok := tengo.ToString(v)
				if !ok {
					return nil, tengo.ErrInvalidArgumentType{Name: fmt.Sprintf("array[%d]", i), Expected: "string", Found: v.TypeName()}
				}
				prompts[i] = s
			}
			responses, err := r.bridge.HandleBatchRequest(prompts)
			if err != nil {
				return &tengo.Error{Value: &tengo.String{Value: err.Error()}}, nil
			}
			result := make([]tengo.Object, len(responses))
			for i, resp := range responses {
				result[i] = &tengo.String{Value: resp}
				r.llmCalls = append(r.llmCalls, LLMCallRecord{Prompt: prompts[i], Response: resp})
			}
			return &tengo.Array{Value: result}, nil
		},
	})

	// string function (type conversion)
	_ = script.Add("string", &tengo.UserFunction{
		Name: "string",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			return &tengo.String{Value: objectToString(args[0])}, nil
		},
	})
}

// extractVariables extracts all global variables from the compiled script
func (r *GoREPL) extractVariables(compiled *tengo.Compiled) {
	// Get all variable names from the compiled script
	for _, v := range compiled.GetAll() {
		name := v.Name()
		if name == "" || name == "context" {
			continue
		}
		r.variables[name] = r.fromTengoObject(v.Object())
	}
}

// GetVariable retrieves a variable by name
func (r *GoREPL) GetVariable(name string) (any, error) {
	val, ok := r.variables[name]
	if !ok {
		return nil, fmt.Errorf("variable %q not found", name)
	}
	return val, nil
}

// SetVariable sets a variable in the REPL state
func (r *GoREPL) SetVariable(name string, value any) error {
	r.variables[name] = value
	return nil
}

// Reset clears all state except injected functions
func (r *GoREPL) Reset() error {
	r.variables = make(map[string]any)
	r.output.Reset()
	r.llmCalls = nil
	r.ctx = ""
	return nil
}

// toTengoValue converts a Go value to a Tengo-compatible value
func (r *GoREPL) toTengoValue(v any) any {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return val
	case bool:
		return val
	case []string:
		return val
	case []any:
		return val
	case map[string]any:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

// fromTengoObject converts a Tengo object to a Go value
func (r *GoREPL) fromTengoObject(obj tengo.Object) any {
	switch v := obj.(type) {
	case *tengo.String:
		return v.Value
	case *tengo.Int:
		return int(v.Value)
	case *tengo.Float:
		return v.Value
	case *tengo.Bool:
		if !v.IsFalsy() {
			return true
		}
		return false
	case *tengo.Array:
		arr := make([]any, len(v.Value))
		for i, item := range v.Value {
			arr[i] = r.fromTengoObject(item)
		}
		return arr
	case *tengo.Map:
		m := make(map[string]any)
		for k, item := range v.Value {
			m[k] = r.fromTengoObject(item)
		}
		return m
	case *tengo.Undefined:
		return nil
	default:
		return obj.String()
	}
}

// objectToString converts a Tengo object to its string representation
func objectToString(obj tengo.Object) string {
	switch v := obj.(type) {
	case *tengo.String:
		return v.Value
	case *tengo.Int:
		return fmt.Sprintf("%d", v.Value)
	case *tengo.Float:
		return fmt.Sprintf("%g", v.Value)
	case *tengo.Bool:
		if !v.IsFalsy() {
			return "true"
		}
		return "false"
	case *tengo.Undefined:
		return "undefined"
	default:
		return obj.String()
	}
}
