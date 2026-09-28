package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrPathTraversal    = errors.New("security: path escapes authorized workspace root")
	ErrInvalidWorkspace = errors.New("security: workspace root does not exist or is invalid")
	ErrEmptyPath        = errors.New("security: target path cannot be empty")
)

// ValidateWorkspacePath resolves targetPath and verifies that it resides strictly
// inside workspaceRoot. It evaluates all symbolic links to defeat symlink directory
// bypass attacks (e.g., symlinks pointing to /etc or /var).
// Returns the canonical, fully-resolved absolute target path.
func ValidateWorkspacePath(workspaceRoot string, targetPath string) (string, error) {
	if workspaceRoot == "" {
		return "", ErrInvalidWorkspace
	}
	if strings.TrimSpace(targetPath) == "" {
		return "", ErrEmptyPath
	}

	cleanRoot := filepath.Clean(workspaceRoot)
	resolvedRoot, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		return "", fmt.Errorf("%w: failed to resolve workspace root: %w", ErrInvalidWorkspace, err)
	}

	absRoot, err := filepath.Abs(resolvedRoot)
	if err != nil {
		return "", fmt.Errorf("%w: failed to determine absolute workspace root: %w", ErrInvalidWorkspace, err)
	}

	var candidate string
	if filepath.IsAbs(targetPath) {
		candidate = filepath.Clean(targetPath)
	} else {
		candidate = filepath.Clean(filepath.Join(absRoot, targetPath))
	}

	// Resolve target path evaluating symlinks
	resolvedTarget, err := evalSymlinksPreservingNonExistent(candidate)
	if err != nil {
		return "", fmt.Errorf("security: failed to resolve target path: %w", err)
	}

	absTarget, err := filepath.Abs(resolvedTarget)
	if err != nil {
		return "", fmt.Errorf("security: failed to determine absolute target path: %w", err)
	}

	// Verify absTarget is contained within absRoot
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", fmt.Errorf("%w: failed to compute relative path: %w", ErrPathTraversal, err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q is outside %q", ErrPathTraversal, absTarget, absRoot)
	}

	return absTarget, nil
}

// evalSymlinksPreservingNonExistent evaluates symlinks for existing path components,
// preserving trailing non-existent components for write operations.
func evalSymlinksPreservingNonExistent(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	// Path does not exist. Traverse upwards until an existing directory is found.
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	if dir == path || dir == "." {
		return path, nil
	}

	resolvedDir, err := evalSymlinksPreservingNonExistent(dir)
	if err != nil {
		return "", err
	}

	return filepath.Join(resolvedDir, base), nil
}
