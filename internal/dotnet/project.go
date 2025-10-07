package dotnet

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

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/util"
)

// projectXML mirrors the root <Project> XML structure.
type projectXML struct {
	XMLName        xml.Name           `xml:"Project"`
	PropertyGroups []propertyGroupXML `xml:"PropertyGroup"`
	ItemGroups     []itemGroupXML     `xml:"ItemGroup"`
}

type itemGroupXML struct {
	PackageReference []packageReferenceXML `xml:"PackageReference"`
	ProjectReference []projectReferenceXML `xml:"ProjectReference"`
}

type packageReferenceXML struct {
	Include     string `xml:"Include,attr"`
	Version     string `xml:"Version,attr"`
	VersionElem string `xml:"Version"`
}

type projectReferenceXML struct {
	Include string `xml:"Include,attr"`
}

type propertyGroupXML struct{ TargetFramework, OutputType, AssemblyName string }

// Project represents a .NET project (.csproj) and its direct project and package references.
type Project struct {
	RootPath          string
	Path              string
	ProjectReferences []Project
	PackageReferences []PackageReference
}

// GetFileName returns the file name (e.g. MyApp.csproj).
func (p Project) GetFileName() string { return filepath.Base(p.Path) }

// GetName returns the project name without extension.
func (p Project) GetName() string { return strings.TrimSuffix(p.GetFileName(), ".csproj") }

// GetRelativePath returns the path relative to the repository root.
func (p Project) GetRelativePath() string {
	return strings.TrimPrefix(strings.TrimPrefix(p.Path, p.RootPath), "/")
}

// GetDirectoryRelativePath returns the containing directory relative path with trailing slash.
func (p Project) GetDirectoryRelativePath() string {
	result := strings.TrimPrefix(strings.TrimPrefix(filepath.Dir(p.Path), p.RootPath), "/")
	if !strings.HasSuffix(result, "/") {
		result += "/"
	}
	return result
}

// GetAllProjectReferences returns a flattened, de-duplicated list of all transitively referenced projects including the root.
func (p Project) GetAllProjectReferences() []Project {
	var result []Project
	seen := map[string]bool{}
	var visit func(Project)
	visit = func(cur Project) {
		if seen[cur.Path] {
			return
		}
		seen[cur.Path] = true
		result = append(result, cur)
		for _, child := range cur.ProjectReferences {
			visit(child)
		}
	}
	visit(p)
	// Case-insensitive sort of paths with original-case tie breaker to ensure deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		li := strings.ToLower(result[i].Path)
		lj := strings.ToLower(result[j].Path)
		if li == lj {
			return result[i].Path < result[j].Path
		}
		return li < lj
	})
	return result
}

// GetProjectReferences returns the direct project references.
func (p Project) GetProjectReferences() []Project { return p.ProjectReferences }

// PackageReference represents a NuGet package reference (Include + Version).
type PackageReference struct{ Include, Version string }

var (
	errMissingProject = fmt.Errorf("missing project file")
	errCircularRef    = fmt.Errorf("circular reference")
)

func innerLoadProject(path string, isMain bool, rootPath string, pathLoaded []string) (Project, error) {
	for _, loadedPath := range pathLoaded {
		if loadedPath == path {
			return Project{}, errCircularRef
		}
	}
	file, err := func(p string) (*os.File, error) {
		// Constrain project file to be within rootPath to mitigate G304 concerns.
		if rootPath != "" {
			absRoot, _ := filepath.Abs(rootPath)
			absPath, _ := filepath.Abs(p)
			if !strings.HasPrefix(absPath, absRoot) {
				return nil, fmt.Errorf("project file outside root: %s", p)
			}
		}
		return os.Open(p) // #nosec G304 - validated above
	}(path)
	if err != nil {
		if !isMain {
			slog.Warn("Cannot open file. Skipped", "path", path, "err", err)
			return Project{}, errMissingProject
		}
		return Project{}, err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(file)
	if err != nil {
		return Project{}, err
	}
	var px projectXML
	if err = xml.Unmarshal(data, &px); err != nil {
		return Project{}, err
	}
	projectReferences := util.SelectMany(util.Where(px.ItemGroups,
		func(ig itemGroupXML) bool { return len(ig.ProjectReference) > 0 }),
		func(ig itemGroupXML) []projectReferenceXML { return ig.ProjectReference })
	var references []Project
	for _, pr := range projectReferences {
		childPath := strings.ReplaceAll(pr.Include, "\\", "/")
		childPath = filepath.Join(filepath.Dir(path), childPath)
		child, prjErr := innerLoadProject(childPath, false, rootPath, append(pathLoaded, path))
		if prjErr != nil {
			if errors.Is(prjErr, errMissingProject) {
				continue
			}
			return Project{}, prjErr
		}
		references = append(references, child)
	}
	packageReferences := util.SelectMany(util.Where(px.ItemGroups,
		func(ig itemGroupXML) bool { return len(ig.PackageReference) > 0 }),
		func(ig itemGroupXML) []packageReferenceXML { return ig.PackageReference })
	var packages []PackageReference
	for _, pr := range packageReferences {
		v := pr.Version
		if v == "" {
			v = pr.VersionElem
		}
		packages = append(packages, PackageReference{Include: pr.Include, Version: v})
	}
	return Project{RootPath: rootPath, Path: path, ProjectReferences: references, PackageReferences: packages}, nil
}

// LoadProject loads a root .csproj and recursively its transitive project references.
func LoadProject(path, rootPath string) (Project, error) {
	return innerLoadProject(path, true, rootPath, []string{})
}
