package cmd

import (
	"github.com/itsmostafa/goralph/internal/loop"
	"github.com/spf13/cobra"
)

var planMaxIterations int

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Run the agentic loop in plan mode",
	Long:  `Run Claude Code in plan mode using .ralph/PROMPT_plan.md as the prompt file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return loop.Run(loop.Config{
			Mode:          loop.ModePlan,
			PromptFile:    ".ralph/PROMPT_plan.md",
			MaxIterations: planMaxIterations,
		})
	},
}

func init() {
	planCmd.Flags().IntVarP(&planMaxIterations, "max", "n", 0, "Maximum number of iterations (0 = unlimited)")
	rootCmd.AddCommand(planCmd)
}
