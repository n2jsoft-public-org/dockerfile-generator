package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ProjectXml struct {
	XMLName        xml.Name           `xml:"Project"`
	PropertyGroups []PropertyGroupXml `xml:"PropertyGroup"`
	ItemGroups     []ItemGroupXml     `xml:"ItemGroup"`
}

type ItemGroupXml struct {
	PackageReference []PackageReferenceXml `xml:"PackageReference"`
	ProjectReference []ProjectReferenceXml `xml:"ProjectReference"`
}

type PackageReferenceXml struct {
	Include string `xml:"Include,attr"`
	Version string `xml:"Version,attr"`
}

type ProjectReferenceXml struct {
	Include string `xml:"Include,attr"`
}

type PropertyGroupXml struct {
	TargetFramework string `xml:"TargetFramework"`
	OutputType      string `xml:"OutputType"`
	AssemblyName    string `xml:"AssemblyName"`
}

type Project struct {
	RootPath          string
	Path              string
	ProjectReferences []Project
	PackageReferences []PackageReference
}

func (p Project) GetFileName() string {
	return filepath.Base(p.Path)
}

func (p Project) GetName() string {
	return strings.TrimSuffix(p.GetFileName(), ".csproj")
}

func (p Project) GetRelativePath() string {
	result := p.Path
	if strings.HasPrefix(result, p.RootPath) {
		result = result[len(p.RootPath):]
	}

	if strings.HasPrefix(result, "/") {
		result = result[1:]
	}

	return result
}

func (p Project) GetDirectoryRelativePath() string {
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
func (p Project) GetAllProjectReferences() []Project {
	var result []Project
	seen := make(map[string]bool)

	result = append(result, p)
	seen[p.Path] = true

	for _, pr := range p.ProjectReferences {
		if len(pr.ProjectReferences) > 0 {
			subRefs := pr.GetAllProjectReferences()
			for _, ref := range subRefs {
				if !seen[ref.Path] {
					result = append(result, ref)
					seen[ref.Path] = true
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

func (p Project) GetProjectReferences() []Project {
	return p.ProjectReferences
}

type PackageReference struct {
	Include string
	Version string
}

var (
	ErrMissingProject = fmt.Errorf("missing project file")
	ErrCircularRef    = fmt.Errorf("circular reference")
)

func innerLoadProject(path string, isMain bool, rootPath string, pathLoaded []string) (Project, error) {
	for _, loadedPath := range pathLoaded {
		if loadedPath == path {
			return Project{}, ErrCircularRef
		}
	}

	file, err := os.Open(path)
	if err != nil {
		if !isMain {
			slog.Warn("Cannot open file. Skipped", path, err)
			return Project{}, ErrMissingProject
		}
		return Project{}, err
	}

	defer func(file *os.File) {
		fileErr := file.Close()
		if fileErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error closing file: %v\n", fileErr)
		}
	}(file)

	data, err := io.ReadAll(file)
	if err != nil {
		return Project{}, err
	}

	var projectXml ProjectXml
	err = xml.Unmarshal(data, &projectXml)
	if err != nil {
		return Project{}, err
	}

	projectReferences := SelectMany(
		Where(projectXml.ItemGroups,
			func(ig ItemGroupXml) bool {
				return ig.ProjectReference != nil && len(ig.ProjectReference) > 0
			}),
		func(ig ItemGroupXml) []ProjectReferenceXml {
			return ig.ProjectReference
		})
	var references []Project
	if len(projectReferences) > 0 {
		for _, pr := range projectReferences {
			childPath := strings.Replace(pr.Include, "\\", "/", -1)
			childPath = filepath.Join(filepath.Dir(path), childPath)
			child, prjErr := innerLoadProject(childPath, false, rootPath, append(pathLoaded, path))
			if prjErr != nil {
				if errors.Is(prjErr, ErrMissingProject) {
					continue
				}
				return Project{}, prjErr
			}
			references = append(references, child)
		}
	}
	packageReferences := SelectMany(
		Where(projectXml.ItemGroups,
			func(ig ItemGroupXml) bool {
				return ig.PackageReference != nil && len(ig.PackageReference) > 0
			}),
		func(ig ItemGroupXml) []PackageReferenceXml {
			return ig.PackageReference
		})
	var packages []PackageReference
	if len(packageReferences) > 0 {
		packages = make([]PackageReference, len(packageReferences))
		for i, pr := range packageReferences {
			packages[i] = PackageReference{
				Include: pr.Include,
				Version: pr.Version,
			}
		}
	}

	return Project{
		RootPath:          rootPath,
		Path:              path,
		ProjectReferences: references,
		PackageReferences: packages,
	}, nil
}
func loadProject(path, rootPath string) (Project, error) {
	return innerLoadProject(path, true, rootPath, []string{})
}
