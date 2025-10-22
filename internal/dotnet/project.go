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

	"github.com/n2jsoft-public-org/dockerfile-generator/internal/util"
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
	// resolve and load all child project references (supports wildcards)
	references, refErr := loadProjectReferences(filepath.Dir(path), projectReferences, rootPath, append(pathLoaded, path))
	if refErr != nil {
		return Project{}, refErr
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

// loadProjectReferences resolves project reference includes (supports wildcards) and loads each child project.
func loadProjectReferences(baseDir string, projectReferences []projectReferenceXML, rootPath string, pathLoaded []string) ([]Project, error) {
	var references []Project
	for _, pr := range projectReferences {
		childPaths := resolveChildPaths(baseDir, pr.Include)
		for _, cp := range childPaths {
			child, prjErr := innerLoadProject(cp, false, rootPath, pathLoaded)
			if prjErr != nil {
				if errors.Is(prjErr, errMissingProject) {
					continue
				}
				return nil, prjErr
			}
			references = append(references, child)
		}
	}
	return references, nil
}

// resolveChildPaths turns a ProjectReference Include into concrete file paths.
// It supports patterns like:
// - "path/**/*.csproj" → recursively find all .csproj under "path"
// - "path/*.csproj"    → find all .csproj directly under "path" (non-recursive)
// - explicit file path  → single result
func resolveChildPaths(baseDir, include string) []string {
	inc := strings.ReplaceAll(include, "\\", "/")

	// Handle recursive pattern ".../**/....csproj"
	if strings.Contains(inc, "/**/") && strings.HasSuffix(strings.ToLower(inc), ".csproj") {
		prefix := inc[:strings.Index(inc, "/**/")]
		startDir := filepath.Join(baseDir, prefix)
		return listCsprojRecursive(startDir)
	}
	// Handle explicit recursive suffix ".../**/.csproj" and ".../**/*.csproj"
	if strings.HasSuffix(inc, "/**/*.csproj") {
		prefix := strings.TrimSuffix(inc, "/**/*.csproj")
		startDir := filepath.Join(baseDir, prefix)
		return listCsprojRecursive(startDir)
	}
	// Handle non-recursive: ".../*.csproj" or just "*.csproj"
	if strings.HasSuffix(inc, "/*.csproj") {
		dirRel := strings.TrimSuffix(inc, "/*.csproj")
		absDir := filepath.Join(baseDir, dirRel)
		return listCsprojInDir(absDir)
	}
	if inc == "*.csproj" {
		return listCsprojInDir(baseDir)
	}

	// Generic glob fallback for other wildcard usages (non-recursive)
	if strings.ContainsAny(inc, "*?[]") {
		pattern := filepath.Join(baseDir, inc)
		matches, _ := filepath.Glob(pattern)
		sort.Strings(matches)
		return matches
	}

	// No wildcard: treat as direct file path
	return []string{filepath.Join(baseDir, inc)}
}

// listCsprojRecursive walks dir recursively and returns all .csproj files.
func listCsprojRecursive(dir string) []string {
	var out []string
	// If directory doesn't exist, nothing to do.
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return out
	}
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info != nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".csproj") {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// listCsprojInDir returns all .csproj directly within dir (non-recursive).
func listCsprojInDir(dir string) []string {
	var out []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".csproj") {
			out = append(out, filepath.Join(dir, name))
		}
	}
	sort.Strings(out)
	return out
}
