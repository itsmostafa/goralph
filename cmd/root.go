package cmd

import (
	"fmt"
	"os"

	"github.com/itsmostafa/goralph/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goralph",
	Short: "Ralph Wiggum agentic loop for Claude Code",
	Long: `Go Ralph is an implementation of the Ralph Wiggum Technique - an agentic loop
pattern that runs Claude Code iteratively with automatic git pushes between iterations.

Reference: https://github.com/ghuntley/how-to-ralph-wiggum`,
}

func init() {
	rootCmd.Version = version.Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("goralph %s\n", version.String()))
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
