package cmd

import (
	"os"

	"github.com/itsmostafa/goralph/internal/loop"
	"github.com/spf13/cobra"
)

var planMaxIterations int
var planNoPush bool
var planCLI string

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Run the agentic loop in plan mode",
	Long:  `Run Claude Code in plan mode using .ralph/PROMPT_plan.md as the prompt file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate CLI provider
		cliProvider, err := loop.ValidateCLIProvider(planCLI)
		if err != nil {
			return err
		}

		return loop.Run(loop.Config{
			Mode:          loop.ModePlan,
			PromptFile:    ".ralph/PROMPT_plan.md",
			MaxIterations: planMaxIterations,
			NoPush:        planNoPush,
			CLI:           cliProvider,
			Output:        cmd.OutOrStdout(),
		})
	},
}

func init() {
	planCmd.Flags().IntVarP(&planMaxIterations, "max", "n", 0, "Maximum number of iterations (0 = unlimited)")
	planCmd.Flags().BoolVar(&planNoPush, "no-push", false, "Skip pushing changes after each iteration")

	// CLI provider flag with env var fallback
	defaultCLI := "claude"
	if envCLI := os.Getenv("GORALPH_CLI"); envCLI != "" {
		defaultCLI = envCLI
	}
	planCmd.Flags().StringVar(&planCLI, "cli", defaultCLI, "CLI provider to use (claude, codex)")

	rootCmd.AddCommand(planCmd)
}
