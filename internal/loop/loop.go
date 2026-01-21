package loop

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Mode represents the execution mode
type Mode string

const (
	ModeBuild Mode = "build"
	ModePlan  Mode = "plan"
)

// Config holds the loop configuration
type Config struct {
	Mode          Mode
	PromptFile    string
	MaxIterations int
}

// Run executes the agentic loop
func Run(cfg Config) error {
	// Get current git branch
	branch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Verify prompt file exists
	if _, err := os.Stat(cfg.PromptFile); os.IsNotExist(err) {
		return fmt.Errorf("prompt file not found: %s", cfg.PromptFile)
	}

	// Print configuration
	printHeader(cfg, branch)

	iteration := 0
	for {
		// Check max iterations
		if cfg.MaxIterations > 0 && iteration >= cfg.MaxIterations {
			fmt.Printf("Reached max iterations: %d\n", cfg.MaxIterations)
			break
		}

		// Run Claude iteration
		if err := runClaudeIteration(cfg.PromptFile); err != nil {
			return fmt.Errorf("claude iteration failed: %w", err)
		}

		// Push changes
		if err := pushChanges(branch); err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}

		iteration++
		fmt.Printf("\n\n======================== LOOP %d ========================\n\n", iteration)
	}

	return nil
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func printHeader(cfg Config, branch string) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Mode:   %s\n", cfg.Mode)
	fmt.Printf("Prompt: %s\n", cfg.PromptFile)
	fmt.Printf("Branch: %s\n", branch)
	if cfg.MaxIterations > 0 {
		fmt.Printf("Max:    %d iterations\n", cfg.MaxIterations)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func runClaudeIteration(promptFile string) error {
	// Read prompt file
	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		return fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Run claude with the prompt piped to stdin
	cmd := exec.Command("claude",
		"-p",
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--verbose",
	)

	// Set up stdin with prompt content
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Connect stdout and stderr to terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude: %w", err)
	}

	// Write prompt to stdin and close
	if _, err := stdin.Write(promptContent); err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}
	stdin.Close()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude exited with error: %w", err)
	}

	return nil
}

func pushChanges(branch string) error {
	// Try to push
	cmd := exec.Command("git", "push", "origin", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If push failed, try to create remote branch
		fmt.Println("Failed to push. Creating remote branch...")
		cmd = exec.Command("git", "push", "-u", "origin", branch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return nil
}
