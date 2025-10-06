package dotnet

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/common"
)

const (
	nugetFileName           = "nuget.config"
	directoryBuildPropsName = "Directory.Build.props"
	directoryPackagesProps  = "Directory.Packages.props"
)

type searchCache struct {
	nugetConfigPath     string
	nugetConfigSearched bool
	directoryFiles      map[string][]string // directory|fileName -> files found
	searchedDirs        map[string]bool
}

func newSearchCache() *searchCache {
	return &searchCache{directoryFiles: map[string][]string{}, searchedDirs: map[string]bool{}}
}

func LoadProjectContextFromProject(project Project, rootPath string) ([]common.AdditionalFilePath, error) {
	var additionalPaths []common.AdditionalFilePath
	seen := map[string]bool{}
	cache := newSearchCache()
	for _, p := range project.GetAllProjectReferences() {
		slog.Info("Looking for project context file", "path", p.Path)
		paths, err := loadProjectContextCached(p.Path, rootPath, cache)
		if err != nil {
			return nil, err
		}
		for _, f := range paths {
			if !seen[f] {
				additionalPaths = append(additionalPaths, common.AdditionalFilePath{Path: f, RootPath: rootPath})
				seen[f] = true
			}
		}
	}
	return additionalPaths, nil
}

func loadProjectContextCached(path, rootPath string, cache *searchCache) ([]string, error) {
	var paths []string
	if !cache.nugetConfigSearched {
		cache.nugetConfigPath = findNugetConfigFile(rootPath)
		cache.nugetConfigSearched = true
	}
	if cache.nugetConfigPath != "" {
		paths = append(paths, cache.nugetConfigPath)
	}
	paths = append(paths, findAllFileMatchingCached(path, rootPath, directoryBuildPropsName, cache)...)
	paths = append(paths, findAllFileMatchingCached(path, rootPath, directoryPackagesProps, cache)...)
	return paths, nil
}

func findAllFileMatchingCached(startAt, rootPath, fileName string, cache *searchCache) []string {
	var result []string
	current := startAt
	for {
		key := current + "|" + fileName
		if files, found := cache.directoryFiles[key]; found {
			result = append(result, files...)
		} else if !cache.searchedDirs[key] {
			cache.searchedDirs[key] = true
			var foundFiles []string
			matches, err := filepath.Glob(filepath.Join(current, "*"+fileName+"*"))
			if err == nil {
				for _, m := range matches {
					if strings.EqualFold(filepath.Base(m), fileName) {
						foundFiles = append(foundFiles, m)
					}
				}
			}
			cache.directoryFiles[key] = foundFiles
			result = append(result, foundFiles...)
		}
		if current == rootPath {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return result
}

func findNugetConfigFile(path string) string {
	var found string
	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(filepath.Base(p), nugetFileName) {
			found = p
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
