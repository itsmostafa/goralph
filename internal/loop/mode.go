package loop

import (
	"fmt"
	"io"
)

// Mode represents the execution mode for the agentic loop
type Mode string

const (
	// ModeRalph is the default ralph mode using implementation plans
	ModeRalph Mode = "ralph"
	// ModeRLM is the RLM (Recursive Language Model) mode with state persistence
	ModeRLM Mode = "rlm"
)

// ValidateMode checks if the given mode string is valid and returns the Mode
func ValidateMode(mode string) (Mode, error) {
	switch Mode(mode) {
	case ModeRalph:
		return ModeRalph, nil
	case ModeRLM:
		return ModeRLM, nil
	default:
		return "", fmt.Errorf("unknown mode: %q (valid options: ralph, rlm)", mode)
	}
}

// RLM marker patterns for output detection
const (
	// RLMPhaseMarkerStart is the start of a phase marker
	RLMPhaseMarkerStart = "<rlm:phase>"
	// RLMPhaseMarkerEnd is the end of a phase marker
	RLMPhaseMarkerEnd = "</rlm:phase>"
	// RLMVerifiedMarkerStart is the start of a verified marker
	RLMVerifiedMarkerStart = "<rlm:verified>"
	// RLMVerifiedMarkerEnd is the end of a verified marker
	RLMVerifiedMarkerEnd = "</rlm:verified>"
)

// BannerInfo contains information for rendering the loop banner
type BannerInfo struct {
	Phase string
}

// ModeRunner defines the interface for mode-specific behavior
type ModeRunner interface {
	// Name returns the mode name for display purposes
	Name() string

	// Initialize sets up mode-specific state at the start of a session
	Initialize(cfg Config) error

	// BuildPrompt constructs the prompt for the given iteration
	BuildPrompt(cfg Config, iteration int) ([]byte, error)

	// HandleResult processes the result from an agent iteration
	HandleResult(cfg Config, result *ResultMessage, iteration int) error

	// GetBannerInfo returns information for rendering the loop banner
	GetBannerInfo() BannerInfo

	// ShouldRunVerification determines if verification should run for this result
	ShouldRunVerification(cfg Config, result *ResultMessage) bool

	// StoreVerification stores the verification report (if applicable)
	StoreVerification(report VerificationReport) error

	// Output returns the writer for mode output
	Output() io.Writer

	// SetOutput sets the writer for mode output
	SetOutput(w io.Writer)
}
