package cmd

import (
	"os"

	"github.com/itsmostafa/goralph/internal/loop"
	"github.com/spf13/cobra"
)

var buildMaxIterations int
var buildNoPush bool
var buildCLI string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run the agentic loop in build mode",
	Long:  `Run Claude Code in build mode using .ralph/PROMPT_build.md as the prompt file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate CLI provider
		cliProvider, err := loop.ValidateCLIProvider(buildCLI)
		if err != nil {
			return err
		}

		return loop.Run(loop.Config{
			Mode:          loop.ModeBuild,
			PromptFile:    ".ralph/PROMPT_build.md",
			MaxIterations: buildMaxIterations,
			NoPush:        buildNoPush,
			CLI:           cliProvider,
			Output:        cmd.OutOrStdout(),
		})
	},
}

func init() {
	buildCmd.Flags().IntVarP(&buildMaxIterations, "max", "n", 0, "Maximum number of iterations (0 = unlimited)")
	buildCmd.Flags().BoolVar(&buildNoPush, "no-push", false, "Skip pushing changes after each iteration")

	// CLI provider flag with env var fallback
	defaultCLI := "claude"
	if envCLI := os.Getenv("GORALPH_CLI"); envCLI != "" {
		defaultCLI = envCLI
	}
	buildCmd.Flags().StringVar(&buildCLI, "cli", defaultCLI, "CLI provider to use (claude, codex)")

	rootCmd.AddCommand(buildCmd)
}
