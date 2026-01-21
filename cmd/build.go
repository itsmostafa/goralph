package cmd

import (
	"github.com/itsmostafa/goralph/internal/loop"
	"github.com/spf13/cobra"
)

var buildMaxIterations int

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run the agentic loop in build mode",
	Long:  `Run Claude Code in build mode using PROMPT_build.md as the prompt file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return loop.Run(loop.Config{
			Mode:          loop.ModeBuild,
			PromptFile:    "PROMPT_build.md",
			MaxIterations: buildMaxIterations,
		})
	},
}

func init() {
	buildCmd.Flags().IntVarP(&buildMaxIterations, "max", "n", 0, "Maximum number of iterations (0 = unlimited)")
	rootCmd.AddCommand(buildCmd)
}
