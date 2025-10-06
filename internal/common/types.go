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

func (p AdditionalFilePath) GetRelativePath() string {
	result := p.Path
	if strings.HasPrefix(result, p.RootPath) {
		result = result[len(p.RootPath):]
	}
	result = strings.TrimPrefix(result, "/")
	return result
}

func (p AdditionalFilePath) GetDirectoryRelativePath() string {
	result := filepath.Dir(p.Path)
	if strings.HasPrefix(result, p.RootPath) {
		result = result[len(p.RootPath):]
	}
	result = strings.TrimPrefix(result, "/")
	if !strings.HasSuffix(result, "/") {
		result += "/"
	}
	return result
}
