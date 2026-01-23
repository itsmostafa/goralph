package rlm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
)

// FSModule provides filesystem functions for the REPL environment.
// These functions allow the agent to explore and load codebase files on demand.
type FSModule struct {
	// WorkDir is the working directory for relative paths (defaults to current directory)
	WorkDir string

	// MaxFileSize is the maximum file size in bytes to read (default: 1MB)
	MaxFileSize int64

	// AllowedExtensions limits which file extensions can be read (empty = all allowed)
	AllowedExtensions []string

	// ExcludeDirs is a list of directory names to exclude from listings
	ExcludeDirs []string
}

// NewFSModule creates a new filesystem module with default settings.
func NewFSModule() *FSModule {
	wd, _ := os.Getwd()
	return &FSModule{
		WorkDir:     wd,
		MaxFileSize: 1024 * 1024, // 1MB default
		ExcludeDirs: []string{".git", "node_modules", "__pycache__", ".venv", "vendor", ".ralph"},
	}
}

// resolvePath converts a path to an absolute path within the working directory.
// It ensures the resolved path is within the working directory for security.
func (f *FSModule) resolvePath(path string) (string, error) {
	// Handle absolute paths
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	// Resolve relative to working directory
	resolved := filepath.Join(f.WorkDir, path)
	return filepath.Clean(resolved), nil
}

// List returns the files and directories in the specified path.
func (f *FSModule) List(path string) ([]map[string]any, error) {
	resolved, err := f.resolvePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for _, entry := range entries {
		// Skip excluded directories
		if entry.IsDir() && f.isExcludedDir(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		result = append(result, map[string]any{
			"name":  entry.Name(),
			"isDir": entry.IsDir(),
			"size":  info.Size(),
		})
	}

	return result, nil
}

// Read reads the contents of a file.
func (f *FSModule) Read(path string) (string, error) {
	resolved, err := f.resolvePath(path)
	if err != nil {
		return "", err
	}

	// Check file size first
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return "", os.ErrInvalid
	}

	if info.Size() > f.MaxFileSize {
		// Read only up to max size
		file, err := os.Open(resolved)
		if err != nil {
			return "", err
		}
		defer file.Close()

		buf := make([]byte, f.MaxFileSize)
		n, err := file.Read(buf)
		if err != nil {
			return "", err
		}
		return string(buf[:n]) + "\n... [truncated]", nil
	}

	content, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// Glob finds files matching a glob pattern.
func (f *FSModule) Glob(pattern string) ([]string, error) {
	// Resolve pattern relative to working directory if not absolute
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(f.WorkDir, pattern)
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Convert to relative paths and filter excluded dirs
	var result []string
	for _, match := range matches {
		// Check if any path component is excluded
		if f.containsExcludedDir(match) {
			continue
		}

		// Convert to relative path if possible
		rel, err := filepath.Rel(f.WorkDir, match)
		if err == nil {
			result = append(result, rel)
		} else {
			result = append(result, match)
		}
	}

	return result, nil
}

// Exists checks if a file or directory exists.
func (f *FSModule) Exists(path string) bool {
	resolved, err := f.resolvePath(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(resolved)
	return err == nil
}

// Tree returns a tree structure of the directory up to the specified depth.
func (f *FSModule) Tree(path string, maxDepth int) ([]map[string]any, error) {
	resolved, err := f.resolvePath(path)
	if err != nil {
		return nil, err
	}

	return f.buildTree(resolved, 0, maxDepth)
}

// buildTree recursively builds the directory tree.
func (f *FSModule) buildTree(path string, currentDepth, maxDepth int) ([]map[string]any, error) {
	if currentDepth >= maxDepth {
		return nil, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for _, entry := range entries {
		// Skip excluded directories
		if entry.IsDir() && f.isExcludedDir(entry.Name()) {
			continue
		}

		node := map[string]any{
			"name":  entry.Name(),
			"isDir": entry.IsDir(),
		}

		if entry.IsDir() {
			children, err := f.buildTree(filepath.Join(path, entry.Name()), currentDepth+1, maxDepth)
			if err == nil && len(children) > 0 {
				node["children"] = children
			}
		}

		result = append(result, node)
	}

	return result, nil
}

// isExcludedDir checks if a directory name should be excluded.
func (f *FSModule) isExcludedDir(name string) bool {
	for _, excluded := range f.ExcludeDirs {
		if name == excluded {
			return true
		}
	}
	return false
}

// containsExcludedDir checks if a path contains any excluded directory.
func (f *FSModule) containsExcludedDir(path string) bool {
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if f.isExcludedDir(part) {
			return true
		}
	}
	return false
}

// SetupFSModule adds the 'fs' object with filesystem functions to the goja VM.
func SetupFSModule(vm *goja.Runtime, fsModule *FSModule) error {
	fs := vm.NewObject()

	// fs.list(path) -> array of {name, isDir, size}
	list := func(call goja.FunctionCall) goja.Value {
		path := "."
		if len(call.Arguments) > 0 {
			path = call.Arguments[0].String()
		}

		result, err := fsModule.List(path)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(result)
	}
	if err := fs.Set("list", list); err != nil {
		return err
	}

	// fs.read(path) -> string content
	read := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("fs.read requires 1 argument: path"))
		}
		path := call.Arguments[0].String()

		content, err := fsModule.Read(path)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(content)
	}
	if err := fs.Set("read", read); err != nil {
		return err
	}

	// fs.glob(pattern) -> array of matching paths
	glob := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("fs.glob requires 1 argument: pattern"))
		}
		pattern := call.Arguments[0].String()

		matches, err := fsModule.Glob(pattern)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(matches)
	}
	if err := fs.Set("glob", glob); err != nil {
		return err
	}

	// fs.exists(path) -> boolean
	exists := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("fs.exists requires 1 argument: path"))
		}
		path := call.Arguments[0].String()

		return vm.ToValue(fsModule.Exists(path))
	}
	if err := fs.Set("exists", exists); err != nil {
		return err
	}

	// fs.tree(path, depth) -> nested array of {name, isDir, children}
	tree := func(call goja.FunctionCall) goja.Value {
		path := "."
		depth := 3 // default depth
		if len(call.Arguments) > 0 {
			path = call.Arguments[0].String()
		}
		if len(call.Arguments) > 1 {
			depth = int(call.Arguments[1].ToInteger())
		}

		result, err := fsModule.Tree(path, depth)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(result)
	}
	if err := fs.Set("tree", tree); err != nil {
		return err
	}

	return vm.Set("fs", fs)
}
