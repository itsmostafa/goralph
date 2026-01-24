package cmd

import (
	"os"

	"github.com/itsmostafa/goralph/internal/loop"
	"github.com/spf13/cobra"
)

var maxIterations int
var noPush bool
var agent string
var rlmEnabled bool
var verifyEnabled bool
var maxDepth int

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the agentic loop",
	Long:  `Run the agentic loop using .ralph/PROMPT.md as the prompt file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate agent provider
		agentProvider, err := loop.ValidateAgentProvider(agent)
		if err != nil {
			return err
		}

		return loop.Run(loop.Config{
			PromptFile:    loop.PromptFile,
			PlanFile:      loop.GeneratePlanPath(),
			MaxIterations: maxIterations,
			NoPush:        noPush,
			Agent:         agentProvider,
			Output:        cmd.OutOrStdout(),
			RLM: loop.RLMConfig{
				Enabled:  rlmEnabled,
				MaxDepth: maxDepth,
			},
			VerifyEnabled: verifyEnabled,
		})
	},
}

func init() {
	runCmd.Flags().IntVarP(&maxIterations, "max", "n", 0, "Maximum number of iterations (0 = unlimited)")
	runCmd.Flags().BoolVar(&noPush, "no-push", false, "Skip committing and pushing changes after each iteration")

	// Agent provider flag with env var fallback
	defaultAgent := "claude"
	if envAgent := os.Getenv("GORALPH_AGENT"); envAgent != "" {
		defaultAgent = envAgent
	}
	runCmd.Flags().StringVar(&agent, "agent", defaultAgent, "Agent provider to use (claude, codex)")

	// RLM mode flags
	runCmd.Flags().BoolVar(&rlmEnabled, "rlm", false, "Enable RLM (Recursive Language Model) mode")
	runCmd.Flags().BoolVar(&verifyEnabled, "verify", false, "Run verification (build/test) before commit")
	runCmd.Flags().IntVar(&maxDepth, "max-depth", 3, "Maximum recursion depth for RLM mode")

	rootCmd.AddCommand(runCmd)
}
