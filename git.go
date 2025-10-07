package main

import (
	"os"
	"path/filepath"
)

// findRepositoryRoot walks upward from startPath (file or directory) until it finds a .git directory
// or reaches the filesystem root. Returns empty string if no repository root is found.
func findRepositoryRoot(startPath string) string {
	// Resolve to absolute path (ignore error; non-critical) and normalize to a directory
	if abs, err := filepath.Abs(startPath); err == nil {
		startPath = abs
	}
	if fi, err := os.Stat(startPath); err == nil && !fi.IsDir() {
		startPath = filepath.Dir(startPath)
	}

	currentPath := startPath
	Debugf("git: starting repository root search from %s", currentPath)
	for {
		Debugf("git: checking for .git in %s", currentPath)
		if hasGitDir(currentPath) {
			Debugf("git: found .git directory at %s", currentPath)
			return currentPath
		}
		if isRootPath(currentPath) {
			Debugf("git: reached filesystem root at %s without finding .git", currentPath)
			return ""
		}
		parent := filepath.Dir(currentPath)
		if parent == currentPath { // safety guard (should not happen beyond root check)
			Debugf("git: parent path same as current (%s); aborting search", currentPath)
			return ""
		}
		currentPath = parent
	}
}

func hasGitDir(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	return err == nil && info.IsDir()
}

func isRootPath(path string) bool {
	return path == filepath.Dir(path)
}
