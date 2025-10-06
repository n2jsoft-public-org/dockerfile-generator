// Package common holds shared simple data types used across generators.
// revive:disable:var-naming - package name 'common' is intentional and descriptive here.
package common

import (
	"path/filepath"
	"strings"
)

// AdditionalFilePath represents an extra file (config, props, etc.) to copy into the build context.
type AdditionalFilePath struct {
	Path     string
	RootPath string
}

// GetRelativePath returns the file path relative to the repository root.
func (p AdditionalFilePath) GetRelativePath() string {
	// Simplify prefix removal (S1017): unconditional TrimPrefix is sufficient
	result := strings.TrimPrefix(strings.TrimPrefix(p.Path, p.RootPath), "/")
	return result
}

// GetDirectoryRelativePath returns the directory part (with trailing slash) relative to repo root.
func (p AdditionalFilePath) GetDirectoryRelativePath() string {
	result := strings.TrimPrefix(strings.TrimPrefix(filepath.Dir(p.Path), p.RootPath), "/")
	if !strings.HasSuffix(result, "/") {
		result += "/"
	}
	return result
}
