package cmd

import (
	"os"

	"github.com/itsmostafa/goralph/internal/loop"
	"github.com/spf13/cobra"
)

var planMaxIterations int
var planNoPush bool
var planAgent string

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Run the agentic loop in plan mode",
	Long:  `Run Claude Code in plan mode using .ralph/PROMPT_plan.md as the prompt file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate agent provider
		agentProvider, err := loop.ValidateAgentProvider(planAgent)
		if err != nil {
			return err
		}

		return loop.Run(loop.Config{
			Mode:          loop.ModePlan,
			PromptFile:    ".ralph/PROMPT_plan.md",
			MaxIterations: planMaxIterations,
			NoPush:        planNoPush,
			Agent:         agentProvider,
			Output:        cmd.OutOrStdout(),
		})
	},
}

func init() {
	planCmd.Flags().IntVarP(&planMaxIterations, "max", "n", 0, "Maximum number of iterations (0 = unlimited)")
	planCmd.Flags().BoolVar(&planNoPush, "no-push", false, "Skip pushing changes after each iteration")

	// Agent provider flag with env var fallback
	defaultAgent := "claude"
	if envAgent := os.Getenv("GORALPH_AGENT"); envAgent != "" {
		defaultAgent = envAgent
	}
	planCmd.Flags().StringVar(&planAgent, "agent", defaultAgent, "Agent provider to use (claude, codex)")

	rootCmd.AddCommand(planCmd)
}
