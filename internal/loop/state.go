package loop

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// StateManager handles RLM state persistence
type StateManager struct {
	baseDir string
}

// NewStateManager creates a new StateManager
func NewStateManager(baseDir string) *StateManager {
	return &StateManager{baseDir: baseDir}
}

// InitSession initializes a new RLM session, cleaning up any previous state
func (sm *StateManager) InitSession() (*SessionState, error) {
	// Clean up previous state
	if err := os.RemoveAll(sm.baseDir); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean previous state: %w", err)
	}

	// Create state directories
	dirs := []string{
		sm.baseDir,
		filepath.Join(sm.baseDir, "search"),
		filepath.Join(sm.baseDir, "narrow"),
		filepath.Join(sm.baseDir, "results"),
		filepath.Join(sm.baseDir, "verification"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create state directory %s: %w", dir, err)
		}
	}

	// Create initial session state
	now := time.Now()
	state := &SessionState{
		SessionID:   uuid.New().String(),
		Iteration:   0,
		Depth:       0,
		Phase:       PhasePlan,
		StartedAt:   now,
		LastUpdated: now,
	}

	if err := sm.SaveSession(state); err != nil {
		return nil, err
	}

	// Create initial context manifest
	context := &ContextManifest{
		Task:        TaskInfo{},
		Codebase:    CodebaseInfo{},
		Discoveries: []Discovery{},
		Focus:       FocusSet{},
		LastUpdated: now,
	}
	if err := sm.saveContext(context); err != nil {
		return nil, err
	}

	return state, nil
}

// LoadSession loads the current session state
func (sm *StateManager) LoadSession() (*SessionState, error) {
	path := filepath.Join(sm.baseDir, "session.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session state: %w", err)
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse session state: %w", err)
	}

	return &state, nil
}

// SaveSession saves the session state
func (sm *StateManager) SaveSession(state *SessionState) error {
	state.LastUpdated = time.Now()
	path := filepath.Join(sm.baseDir, "session.json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session state: %w", err)
	}

	return nil
}

// GetContext loads the context manifest
func (sm *StateManager) GetContext() (*ContextManifest, error) {
	path := filepath.Join(sm.baseDir, "context.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read context manifest: %w", err)
	}

	var context ContextManifest
	if err := json.Unmarshal(data, &context); err != nil {
		return nil, fmt.Errorf("failed to parse context manifest: %w", err)
	}

	return &context, nil
}

// UpdateContext updates the context manifest
func (sm *StateManager) UpdateContext(context *ContextManifest) error {
	return sm.saveContext(context)
}

func (sm *StateManager) saveContext(context *ContextManifest) error {
	context.LastUpdated = time.Now()
	path := filepath.Join(sm.baseDir, "context.json")

	data, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write context manifest: %w", err)
	}

	return nil
}

// AppendHistory appends a history entry to the history file
func (sm *StateManager) AppendHistory(entry HistoryEntry) error {
	entry.Timestamp = time.Now()
	path := filepath.Join(sm.baseDir, "history.jsonl")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal history entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write history entry: %w", err)
	}

	return nil
}

// GetHistory reads all history entries
func (sm *StateManager) GetHistory() ([]HistoryEntry, error) {
	path := filepath.Join(sm.baseDir, "history.jsonl")

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed entries
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	return entries, nil
}

// GetRecentHistory returns the last n history entries
func (sm *StateManager) GetRecentHistory(n int) ([]HistoryEntry, error) {
	entries, err := sm.GetHistory()
	if err != nil {
		return nil, err
	}

	if len(entries) <= n {
		return entries, nil
	}
	return entries[len(entries)-n:], nil
}

// StoreSearchResult stores a search query and its results
func (sm *StateManager) StoreSearchResult(query SearchQuery) error {
	query.Timestamp = time.Now()
	filename := fmt.Sprintf("search_%d_%d.json", query.Iteration, time.Now().UnixMilli())
	path := filepath.Join(sm.baseDir, "search", filename)

	data, err := json.MarshalIndent(query, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal search result: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write search result: %w", err)
	}

	return nil
}

// StoreNarrowedSet stores a narrowed context set
func (sm *StateManager) StoreNarrowedSet(set NarrowedSet) error {
	set.Timestamp = time.Now()
	filename := fmt.Sprintf("narrow_%d_%d.json", set.Iteration, time.Now().UnixMilli())
	path := filepath.Join(sm.baseDir, "narrow", filename)

	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal narrowed set: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write narrowed set: %w", err)
	}

	return nil
}

// StoreResult stores an intermediate computation result
func (sm *StateManager) StoreResult(name string, data any) error {
	filename := fmt.Sprintf("%s_%d.json", name, time.Now().UnixMilli())
	path := filepath.Join(sm.baseDir, "results", filename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	return nil
}

// StoreVerification stores a verification report
func (sm *StateManager) StoreVerification(report VerificationReport) error {
	report.Timestamp = time.Now()
	filename := fmt.Sprintf("verify_%d_%d.json", report.Iteration, time.Now().UnixMilli())
	path := filepath.Join(sm.baseDir, "verification", filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal verification report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write verification report: %w", err)
	}

	return nil
}

// GetLatestVerification returns the most recent verification report
func (sm *StateManager) GetLatestVerification() (*VerificationReport, error) {
	dir := filepath.Join(sm.baseDir, "verification")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read verification directory: %w", err)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	// Find most recent file (files are timestamped, so last alphabetically is most recent)
	var latest string
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() > latest {
			latest = entry.Name()
		}
	}

	if latest == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filepath.Join(dir, latest))
	if err != nil {
		return nil, fmt.Errorf("failed to read verification report: %w", err)
	}

	var report VerificationReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse verification report: %w", err)
	}

	return &report, nil
}

// AddDiscovery adds a discovery to the context manifest
func (sm *StateManager) AddDiscovery(discovery Discovery) error {
	context, err := sm.GetContext()
	if err != nil {
		return err
	}

	discovery.Timestamp = time.Now()
	context.Discoveries = append(context.Discoveries, discovery)

	return sm.UpdateContext(context)
}

// UpdateFocus updates the focus set in the context manifest
func (sm *StateManager) UpdateFocus(focus FocusSet) error {
	context, err := sm.GetContext()
	if err != nil {
		return err
	}

	context.Focus = focus
	return sm.UpdateContext(context)
}

// UpdateIteration increments the iteration counter and optionally changes phase
func (sm *StateManager) UpdateIteration(phase Phase) error {
	state, err := sm.LoadSession()
	if err != nil {
		return err
	}

	state.Iteration++
	state.Phase = phase

	return sm.SaveSession(state)
}
