package loop

import (
	"bytes"
	"strings"
	"testing"

	"github.com/itsmostafa/goralph/internal/rlm"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name      string
		agent     AgentProvider
		wantName  string
		wantError bool
	}{
		{
			name:      "claude provider",
			agent:     AgentClaude,
			wantName:  "claude",
			wantError: false,
		},
		{
			name:      "codex provider",
			agent:     AgentCodex,
			wantName:  "codex",
			wantError: false,
		},
		{
			name:      "rlm provider",
			agent:     AgentRLM,
			wantName:  "rlm",
			wantError: false,
		},
		{
			name:      "unknown provider",
			agent:     AgentProvider("unknown"),
			wantName:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.agent)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewProvider() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewProvider() unexpected error: %v", err)
			}

			if provider.Name() != tt.wantName {
				t.Errorf("Provider.Name() = %q, want %q", provider.Name(), tt.wantName)
			}
		})
	}
}

func TestClaudeProvider(t *testing.T) {
	provider := &ClaudeProvider{}

	t.Run("Name", func(t *testing.T) {
		if provider.Name() != "claude" {
			t.Errorf("Name() = %q, want %q", provider.Name(), "claude")
		}
	})

	t.Run("Model", func(t *testing.T) {
		model := provider.Model()
		if model == "" {
			t.Error("Model() returned empty string")
		}
		if !strings.Contains(model, "claude") {
			t.Errorf("Model() = %q, expected to contain 'claude'", model)
		}
	})

	t.Run("BuildCommand", func(t *testing.T) {
		prompt := []byte("test prompt")
		cmd, err := provider.BuildCommand(prompt)
		if err != nil {
			t.Fatalf("BuildCommand() error: %v", err)
		}
		if cmd == nil {
			t.Fatal("BuildCommand() returned nil command")
		}
		// Verify command path ends with "claude"
		if !strings.HasSuffix(cmd.Path, "claude") && !strings.Contains(cmd.Args[0], "claude") {
			t.Errorf("BuildCommand() command should be 'claude', got: %v", cmd.Args)
		}
	})
}

func TestCodexProvider(t *testing.T) {
	provider := &CodexProvider{}

	t.Run("Name", func(t *testing.T) {
		if provider.Name() != "codex" {
			t.Errorf("Name() = %q, want %q", provider.Name(), "codex")
		}
	})

	t.Run("Model", func(t *testing.T) {
		model := provider.Model()
		if model == "" {
			t.Error("Model() returned empty string")
		}
		if !strings.Contains(model, "codex") {
			t.Errorf("Model() = %q, expected to contain 'codex'", model)
		}
	})

	t.Run("BuildCommand", func(t *testing.T) {
		prompt := []byte("test prompt")
		cmd, err := provider.BuildCommand(prompt)
		if err != nil {
			t.Fatalf("BuildCommand() error: %v", err)
		}
		if cmd == nil {
			t.Fatal("BuildCommand() returned nil command")
		}
		// Verify command includes "codex"
		if !strings.HasSuffix(cmd.Path, "codex") && !strings.Contains(cmd.Args[0], "codex") {
			t.Errorf("BuildCommand() command should be 'codex', got: %v", cmd.Args)
		}
	})
}

func TestRLMProvider(t *testing.T) {
	config := rlm.DefaultConfig()
	provider := NewRLMProvider(config)

	t.Run("Name", func(t *testing.T) {
		if provider.Name() != "rlm" {
			t.Errorf("Name() = %q, want %q", provider.Name(), "rlm")
		}
	})

	t.Run("Model", func(t *testing.T) {
		model := provider.Model()
		if model == "" {
			t.Error("Model() returned empty string")
		}
		if !strings.Contains(model, "rlm") {
			t.Errorf("Model() = %q, expected to contain 'rlm'", model)
		}
		// Should also contain the underlying model name
		if !strings.Contains(model, config.Model) {
			t.Errorf("Model() = %q, expected to contain underlying model %q", model, config.Model)
		}
	})

	t.Run("BuildCommand returns error", func(t *testing.T) {
		prompt := []byte("test prompt")
		cmd, err := provider.BuildCommand(prompt)
		// RLM uses direct execution, so BuildCommand should return an error
		if err == nil {
			t.Error("BuildCommand() expected error for RLM provider")
		}
		if cmd != nil {
			t.Error("BuildCommand() expected nil command for RLM provider")
		}
		if !strings.Contains(err.Error(), "direct execution") {
			t.Errorf("BuildCommand() error should mention direct execution, got: %v", err)
		}
	})

	t.Run("ParseOutput returns error", func(t *testing.T) {
		_, err := provider.ParseOutput(nil, nil, nil)
		if err == nil {
			t.Error("ParseOutput() expected error for RLM provider")
		}
		if !strings.Contains(err.Error(), "RunDirect") {
			t.Errorf("ParseOutput() error should mention RunDirect, got: %v", err)
		}
	})

	t.Run("implements DirectRunner", func(t *testing.T) {
		// Verify RLMProvider implements DirectRunner interface
		var _ DirectRunner = provider
	})
}

func TestRLMProviderDirectRunner(t *testing.T) {
	// Skip integration tests that require API calls
	t.Skip("Skipping RLM DirectRunner integration test - requires API key")

	config := rlm.DefaultConfig()
	provider := NewRLMProvider(config)

	var output bytes.Buffer
	var logFile bytes.Buffer

	prompt := []byte("What is 2 + 2?")
	result, err := provider.RunDirect(prompt, &output, &logFile)
	if err != nil {
		t.Fatalf("RunDirect() error: %v", err)
	}

	if result == nil {
		t.Fatal("RunDirect() returned nil result")
	}

	if result.Type != "result" {
		t.Errorf("RunDirect() result.Type = %q, want %q", result.Type, "result")
	}
}

func TestValidateAgentProvider(t *testing.T) {
	tests := []struct {
		name      string
		agent     string
		want      AgentProvider
		wantError bool
	}{
		{
			name:      "claude",
			agent:     "claude",
			want:      AgentClaude,
			wantError: false,
		},
		{
			name:      "codex",
			agent:     "codex",
			want:      AgentCodex,
			wantError: false,
		},
		{
			name:      "rlm",
			agent:     "rlm",
			want:      AgentRLM,
			wantError: false,
		},
		{
			name:      "invalid",
			agent:     "invalid",
			want:      "",
			wantError: true,
		},
		{
			name:      "empty",
			agent:     "",
			want:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAgentProvider(tt.agent)
			if tt.wantError {
				if err == nil {
					t.Error("ValidateAgentProvider() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateAgentProvider() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ValidateAgentProvider() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewStreamState(t *testing.T) {
	state := NewStreamState()

	if state == nil {
		t.Fatal("NewStreamState() returned nil")
	}
	if state.ActiveTools == nil {
		t.Error("NewStreamState() ActiveTools map is nil")
	}
	if state.CompletedTools == nil {
		t.Error("NewStreamState() CompletedTools map is nil")
	}
	if state.LastTextLen != 0 {
		t.Errorf("NewStreamState() LastTextLen = %d, want 0", state.LastTextLen)
	}
}
