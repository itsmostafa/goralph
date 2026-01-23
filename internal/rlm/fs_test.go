package rlm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFSModule_List(t *testing.T) {
	// Create a temp directory with some files
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("package main"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	fs := NewFSModule()
	fs.WorkDir = tmpDir

	result, err := fs.List(".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 entries, got %d", len(result))
	}

	// Check that we have expected entries
	names := make(map[string]bool)
	for _, entry := range result {
		names[entry["name"].(string)] = true
	}

	if !names["file1.txt"] || !names["file2.go"] || !names["subdir"] {
		t.Errorf("expected file1.txt, file2.go, subdir; got %v", names)
	}
}

func TestFSModule_ListExcludesGit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .git directory (should be excluded)
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	fs := NewFSModule()
	fs.WorkDir = tmpDir

	result, err := fs.List(".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have main.go, not .git
	if len(result) != 1 {
		t.Errorf("expected 1 entry (excluding .git), got %d", len(result))
	}

	if result[0]["name"] != "main.go" {
		t.Errorf("expected main.go, got %s", result[0]["name"])
	}
}

func TestFSModule_Read(t *testing.T) {
	tmpDir := t.TempDir()

	content := "Hello, World!\nLine 2\n"
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(content), 0644)

	fs := NewFSModule()
	fs.WorkDir = tmpDir

	result, err := fs.Read("test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}
}

func TestFSModule_ReadTruncatesLargeFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file larger than MaxFileSize
	largeContent := strings.Repeat("x", 2000)
	os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(largeContent), 0644)

	fs := NewFSModule()
	fs.WorkDir = tmpDir
	fs.MaxFileSize = 100 // Set small limit for testing

	result, err := fs.Read("large.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[truncated]") {
		t.Error("expected truncation marker in output")
	}

	if len(result) > 150 { // 100 bytes + truncation message
		t.Errorf("result should be truncated, got length %d", len(result))
	}
}

func TestFSModule_Glob(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "util.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# readme"), 0644)

	fs := NewFSModule()
	fs.WorkDir = tmpDir

	result, err := fs.Glob("*.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 .go files, got %d", len(result))
	}

	// Verify all results are .go files
	for _, path := range result {
		if !strings.HasSuffix(path, ".go") {
			t.Errorf("expected .go file, got %s", path)
		}
	}
}

func TestFSModule_Exists(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "exists.txt"), []byte("hello"), 0644)

	fs := NewFSModule()
	fs.WorkDir = tmpDir

	if !fs.Exists("exists.txt") {
		t.Error("expected exists.txt to exist")
	}

	if fs.Exists("nonexistent.txt") {
		t.Error("expected nonexistent.txt to not exist")
	}
}

func TestFSModule_Tree(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	os.MkdirAll(filepath.Join(tmpDir, "src", "utils"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "app.go"), []byte("package src"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "utils", "helper.go"), []byte("package utils"), 0644)

	fs := NewFSModule()
	fs.WorkDir = tmpDir

	result, err := fs.Tree(".", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have main.go and src at top level
	if len(result) < 2 {
		t.Errorf("expected at least 2 entries, got %d", len(result))
	}

	// Find src directory
	var srcNode map[string]any
	for _, node := range result {
		if node["name"] == "src" {
			srcNode = node
			break
		}
	}

	if srcNode == nil {
		t.Fatal("expected to find src directory")
	}

	if !srcNode["isDir"].(bool) {
		t.Error("expected src to be a directory")
	}
}

func TestREPLExecutor_FSIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("test data content"), 0644)

	env := NewEnvironment("context content", "test query")
	env.FS.WorkDir = tmpDir

	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	// Test fs.exists
	result := executor.Execute(context.Background(), `fs.exists("data.txt")`)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "true") {
		t.Errorf("expected true, got: %s", result.Output)
	}

	// Test fs.read
	result = executor.Execute(context.Background(), `fs.read("data.txt")`)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "test data content") {
		t.Errorf("expected file content, got: %s", result.Output)
	}
}

func TestREPLExecutor_FSList(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644)

	env := NewEnvironment("", "")
	env.FS.WorkDir = tmpDir

	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), `fs.list(".").length`)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "2") {
		t.Errorf("expected 2 files, got: %s", result.Output)
	}
}

func TestREPLExecutor_FSGlob(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# readme"), 0644)

	env := NewEnvironment("", "")
	env.FS.WorkDir = tmpDir

	config := DefaultConfig()
	executor := NewREPLExecutor(env, config)

	result := executor.Execute(context.Background(), `fs.glob("*.go").length`)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "2") {
		t.Errorf("expected 2 .go files, got: %s", result.Output)
	}
}
