package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	nugetFileName           = "nuget.config"
	directoryBuildPropsName = "Directory.Build.props"
	directoryPackagesProps  = "Directory.Packages.props"
)

type AdditionalFilePath struct {
	Path     string
	RootPath string
}

func (p AdditionalFilePath) GetRelativePath() string {
	result := p.Path
	if strings.HasPrefix(result, p.RootPath) {
		result = result[len(p.RootPath):]
	}

	if strings.HasPrefix(result, "/") {
		result = result[1:]
	}

	return result
}

func (p AdditionalFilePath) GetDirectoryRelativePath() string {
	result := filepath.Dir(p.Path)
	if strings.HasPrefix(result, p.RootPath) {
		result = result[len(p.RootPath):]
	}

	if strings.HasPrefix(result, "/") {
		result = result[1:]
	}
	if !strings.HasSuffix(result, "/") {
		result = result + "/"
	}

	return result
}

type searchCache struct {
	nugetConfigPath     string
	nugetConfigSearched bool
	directoryFiles      map[string][]string // directory -> files found
	searchedDirs        map[string]bool     // directory -> searched
}

func newSearchCache() *searchCache {
	return &searchCache{
		directoryFiles: make(map[string][]string),
		searchedDirs:   make(map[string]bool),
	}
}

func loadProjectContextFromProject(project Project, rootPath string) ([]AdditionalFilePath, error) {
	var additionalPaths []AdditionalFilePath
	seen := make(map[string]bool)
	cache := newSearchCache()

	for _, path := range project.GetAllProjectReferences() {
		slog.Info("Looking for project context file in", "path", path.Path)
		paths, err := loadProjectContextCached(path.Path, rootPath, cache)
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			if !seen[p] {
				additionalPaths = append(additionalPaths, AdditionalFilePath{
					Path:     p,
					RootPath: rootPath,
				})
				seen[p] = true
			}
		}
	}

	return additionalPaths, nil
}

func loadProjectContextCached(path, rootPath string, cache *searchCache) ([]string, error) {
	var paths []string

	// Find nuget config file only once per root path
	if !cache.nugetConfigSearched {
		cache.nugetConfigPath = findNugetConfigFile(rootPath)
		cache.nugetConfigSearched = true
	}
	if cache.nugetConfigPath != "" {
		paths = append(paths, cache.nugetConfigPath)
	}

	directoryBuildPropsPaths := findAllFileMatchingCached(path, rootPath, directoryBuildPropsName, cache)
	if len(directoryBuildPropsPaths) > 0 {
		paths = append(paths, directoryBuildPropsPaths...)
	}
	directoryPackagesPropsPaths := findAllFileMatchingCached(path, rootPath, directoryPackagesProps, cache)
	if len(directoryPackagesPropsPaths) > 0 {
		paths = append(paths, directoryPackagesPropsPaths...)
	}
	return paths, nil
}

func findAllFileMatchingCached(startAt, rootPath, fileName string, cache *searchCache) []string {
	var result []string
	currentPath := startAt

	for {
		// Check if we've already searched this directory for files
		cacheKey := currentPath + "|" + fileName
		if files, found := cache.directoryFiles[cacheKey]; found {
			result = append(result, files...)
		} else if !cache.searchedDirs[cacheKey] {
			// Mark this directory as searched for this file type
			cache.searchedDirs[cacheKey] = true

			var foundFiles []string
			matches, err := filepath.Glob(filepath.Join(currentPath, "*"+fileName+"*"))
			if err == nil {
				for _, match := range matches {
					if strings.EqualFold(filepath.Base(match), fileName) {
						foundFiles = append(foundFiles, match)
					}
				}
			}

			// Cache the results
			cache.directoryFiles[cacheKey] = foundFiles
			result = append(result, foundFiles...)
		}

		if currentPath == rootPath {
			break
		}

		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			break
		}
		currentPath = parentPath
	}

	return result
}

func findNugetConfigFile(path string) string {
	var foundPath string
	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(filepath.Base(p), nugetFileName) {
			foundPath = p
			return filepath.SkipAll
		}
		return nil
	})
	return foundPath
}
