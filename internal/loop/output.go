package loop

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
)

var (
	// titleStyle for bold red headers
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("160"))

	// dimStyle for muted metadata text
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	// successStyle for success indicators
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	// errorStyle for error indicators
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	// boxStyle for summary box with rounded border
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("160")).
			Padding(0, 1)

	// headerBoxStyle for the header
	headerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("160")).
			Padding(0, 1)

	// loopBannerStyle for iteration banners
	loopBannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("160")).
			Padding(0, 2)

	// toolNameStyle for tool names in streaming output
	toolNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

	// toolActiveStyle for the active tool indicator
	toolActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	// toolCompleteStyle for the completed tool indicator
	toolCompleteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))
)

// FormatHeader renders the loop header with configuration info
func FormatHeader(w io.Writer, cfg Config, branch string) {
	var maxLine string
	if cfg.MaxIterations > 0 {
		maxLine = fmt.Sprintf("\n%s %d iterations", dimStyle.Render("Max:"), cfg.MaxIterations)
	}

	// Display agent provider (default to "claude" if not set)
	agentName := string(cfg.Agent)
	if agentName == "" {
		agentName = "claude"
	}

	content := fmt.Sprintf("%s %s  %s %s\n%s %s\n%s %s%s",
		dimStyle.Render("Mode:"), titleStyle.Render(string(cfg.Mode)),
		dimStyle.Render("Agent:"), titleStyle.Render(agentName),
		dimStyle.Render("Prompt:"), cfg.PromptFile,
		dimStyle.Render("Branch:"), successStyle.Render(branch),
		maxLine,
	)

	fmt.Fprintln(w, headerBoxStyle.Render(content))
}

// FormatIterationSummary renders the iteration summary box
func FormatIterationSummary(w io.Writer, result ResultMessage) {
	duration := float64(result.DurationMs) / 1000.0

	// Format token counts with commas
	inputTokens := formatNumber(result.Usage.InputTokens)
	outputTokens := formatNumber(result.Usage.OutputTokens)

	// Build status indicator
	var statusIndicator string
	if result.IsError {
		statusIndicator = errorStyle.Render("ERROR")
	} else {
		statusIndicator = successStyle.Render("OK")
	}

	// Format cost conditionally based on provider support
	var costStr string
	if result.HasCost {
		costStr = fmt.Sprintf("$%.4f", result.TotalCostUSD)
	} else {
		costStr = dimStyle.Render("N/A")
	}

	// Build the summary content
	line1 := fmt.Sprintf("%s %.1fs  %s %d  %s %s",
		dimStyle.Render("Duration:"), duration,
		dimStyle.Render("Turns:"), result.NumTurns,
		dimStyle.Render("Cost:"), costStr,
	)

	line2 := fmt.Sprintf("%s %s in %s %s out  %s",
		dimStyle.Render("Tokens:"), inputTokens,
		dimStyle.Render("->"), outputTokens,
		statusIndicator,
	)

	// Note: Result text is not printed here because it's streamed in real-time
	// during parseClaudeOutput via processAssistantMessage

	// Combine and render summary box after the response
	content := titleStyle.Render("Iteration Complete") + "\n" + line1 + "\n" + line2
	fmt.Fprintln(w, boxStyle.Render(content))
}

// FormatLoopBanner renders the loop iteration banner
func FormatLoopBanner(w io.Writer, iteration int) {
	banner := fmt.Sprintf(" LOOP %d ", iteration)
	fmt.Fprintln(w)
	fmt.Fprintln(w, loopBannerStyle.Render(banner))
	fmt.Fprintln(w)
}

// FormatMaxIterations renders the max iterations reached message
func FormatMaxIterations(w io.Writer, max int) {
	msg := fmt.Sprintf("Reached max iterations: %d", max)
	fmt.Fprintln(w, dimStyle.Render(msg))
}

// formatNumber adds commas to large numbers for readability
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}

// FormatTextDelta writes text content to the output
func FormatTextDelta(w io.Writer, text string) {
	fmt.Fprint(w, text)
}

// FormatToolStart writes a tool invocation indicator
func FormatToolStart(w io.Writer, toolName string) {
	indicator := toolActiveStyle.Render("●")
	name := toolNameStyle.Render(toolName)
	fmt.Fprintf(w, "\n%s %s running...\n", indicator, name)
}

// FormatToolComplete writes a tool completion indicator
func FormatToolComplete(w io.Writer, toolName string) {
	indicator := toolCompleteStyle.Render("✓")
	name := toolNameStyle.Render(toolName)
	fmt.Fprintf(w, "%s %s done\n", indicator, name)
}
