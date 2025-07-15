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

type ProjectContext struct {
	AdditionalFilePaths []string
}

func loadProjectContextFromProject(project Project, rootPath string) ([]string, error) {
	var additionalPaths []string
	seen := make(map[string]bool)
	for _, path := range project.GetAllProjectReferences() {
		slog.Info("Looking for project context file in", "path", path.Path)
		paths, err := loadProjectContext(path.Path, rootPath)
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			if !seen[p] {
				additionalPaths = append(additionalPaths, p)
				seen[p] = true
			}
		}
	}

	return additionalPaths, nil
}

func loadProjectContext(path, rootPath string) ([]string, error) {
	var paths []string
	nugetPath := findNugetConfigFile(rootPath)
	if nugetPath != "" {
		paths = append(paths, nugetPath)
	}
	directoryBuildPropsPaths := findAllFileMatching(path, rootPath, directoryBuildPropsName)
	if len(directoryBuildPropsPaths) > 0 {
		paths = append(paths, directoryBuildPropsPaths...)
	}
	directoryPackagesPropsPaths := findAllFileMatching(path, rootPath, directoryPackagesProps)
	if len(directoryPackagesPropsPaths) > 0 {
		paths = append(paths, directoryPackagesPropsPaths...)
	}
	return paths, nil
}

func findAllFileMatching(startAt, rootPath, fileName string) []string {
	var result []string
	currentPath := startAt

	for {
		matches, err := filepath.Glob(filepath.Join(currentPath, "*"+fileName+"*"))
		if err == nil {
			for _, match := range matches {
				if strings.EqualFold(filepath.Base(match), fileName) {
					result = append(result, match)
				}
			}
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
