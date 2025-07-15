package main

import (
	"os"
	"path/filepath"
)

func findRepositoryRoot(startPath string) string {
	currentPath := startPath
	for {
		if hasGitDir(currentPath) {
			return currentPath
		}
		if isRootPath(currentPath) {
			return ""
		}
		currentPath = filepath.Dir(currentPath)
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
