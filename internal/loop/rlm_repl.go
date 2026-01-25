package loop

// REPLExecutor defines the interface for executing code in a REPL environment
type REPLExecutor interface {
	// Initialize sets up the REPL environment with injected functions
	Initialize() error

	// SetContext loads context into the REPL as the "context" variable
	SetContext(context string) error

	// Execute runs code and returns the result
	Execute(code string) (*REPLResult, error)

	// GetVariable retrieves a variable by name from the REPL state
	GetVariable(name string) (any, error)

	// SetVariable sets a variable in the REPL state
	SetVariable(name string, value any) error

	// Reset clears all state except injected functions
	Reset() error
}
