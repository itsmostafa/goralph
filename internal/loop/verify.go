package loop

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Verifier runs verification commands before commit
type Verifier struct {
	commands []string
}

// NewVerifier creates a new Verifier with the specified commands
// If commands is empty, it will auto-detect project type
func NewVerifier(commands []string) *Verifier {
	if len(commands) == 0 {
		commands = DetectProjectType()
	}
	return &Verifier{commands: commands}
}

// Run executes all verification commands and returns a report
func (v *Verifier) Run(iteration int) VerificationReport {
	report := VerificationReport{
		Iteration: iteration,
		Passed:    true,
		Checks:    make([]VerificationCheck, 0, len(v.commands)),
		Timestamp: time.Now(),
	}

	for _, cmd := range v.commands {
		check := v.runCheck(cmd)
		report.Checks = append(report.Checks, check)
		if !check.Passed {
			report.Passed = false
		}
	}

	return report
}

// runCheck executes a single verification command
func (v *Verifier) runCheck(cmdStr string) VerificationCheck {
	check := VerificationCheck{
		Name:    cmdStr,
		Command: cmdStr,
	}

	// Parse command - simple split on spaces
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		check.Error = "empty command"
		return check
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	check.Output = stdout.String()
	if stderr.Len() > 0 {
		if check.Output != "" {
			check.Output += "\n"
		}
		check.Output += stderr.String()
	}

	if err != nil {
		check.Passed = false
		check.Error = err.Error()
	} else {
		check.Passed = true
	}

	return check
}

// HasCommands returns true if the verifier has commands to run
func (v *Verifier) HasCommands() bool {
	return len(v.commands) > 0
}

// DetectProjectType analyzes the current directory and returns appropriate verification commands
func DetectProjectType() []string {
	// Check for Go project
	if _, err := os.Stat("go.mod"); err == nil {
		return []string{
			"go build ./...",
			"go test ./...",
		}
	}

	// Check for Node.js project
	if _, err := os.Stat("package.json"); err == nil {
		commands := []string{}
		// Check if build script exists
		if hasPkgScript("build") {
			commands = append(commands, "npm run build")
		}
		// Check if test script exists
		if hasPkgScript("test") {
			commands = append(commands, "npm test")
		}
		if len(commands) > 0 {
			return commands
		}
	}

	// Check for Rust project
	if _, err := os.Stat("Cargo.toml"); err == nil {
		return []string{
			"cargo build",
			"cargo test",
		}
	}

	// Check for Python project
	if _, err := os.Stat("pyproject.toml"); err == nil {
		return []string{"pytest"}
	}
	if _, err := os.Stat("setup.py"); err == nil {
		return []string{"pytest"}
	}

	// Check for Makefile
	if _, err := os.Stat("Makefile"); err == nil {
		// Check for common targets
		if hasMakeTarget("test") {
			return []string{"make test"}
		}
		if hasMakeTarget("check") {
			return []string{"make check"}
		}
	}

	// No recognized project type
	return nil
}

// hasPkgScript checks if package.json contains a specific script
func hasPkgScript(script string) bool {
	data, err := os.ReadFile("package.json")
	if err != nil {
		return false
	}
	// Simple check - look for "script": in the JSON
	searchStr := fmt.Sprintf(`"%s":`, script)
	return bytes.Contains(data, []byte(searchStr))
}

// hasMakeTarget checks if Makefile contains a specific target
func hasMakeTarget(target string) bool {
	data, err := os.ReadFile("Makefile")
	if err != nil {
		return false
	}
	// Look for target: at the start of a line
	searchStr := target + ":"
	for _, line := range bytes.Split(data, []byte("\n")) {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte(searchStr)) {
			return true
		}
	}
	return false
}
