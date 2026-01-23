package cmd

import (
	"os"

	"github.com/itsmostafa/goralph/internal/loop"
	"github.com/spf13/cobra"
)

var buildMaxIterations int
var buildNoPush bool
var buildAgent string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run the agentic loop in build mode",
	Long:  `Run the agentic loop in build mode using .ralph/PROMPT.md as the prompt file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate agent provider
		agentProvider, err := loop.ValidateAgentProvider(buildAgent)
		if err != nil {
			return err
		}

		return loop.Run(loop.Config{
			Mode:          loop.ModeBuild,
			PromptFile:    loop.PromptFile,
			PlanFile:      loop.GeneratePlanPath(),
			MaxIterations: buildMaxIterations,
			NoPush:        buildNoPush,
			Agent:         agentProvider,
			Output:        cmd.OutOrStdout(),
		})
	},
}

func init() {
	buildCmd.Flags().IntVarP(&buildMaxIterations, "max", "n", 0, "Maximum number of iterations (0 = unlimited)")
	buildCmd.Flags().BoolVar(&buildNoPush, "no-push", false, "Skip pushing changes after each iteration")

	// Agent provider flag with env var fallback
	defaultAgent := "claude"
	if envAgent := os.Getenv("GORALPH_AGENT"); envAgent != "" {
		defaultAgent = envAgent
	}
	buildCmd.Flags().StringVar(&buildAgent, "agent", defaultAgent, "Agent provider to use (claude, codex, rlm)")

	rootCmd.AddCommand(buildCmd)
}
