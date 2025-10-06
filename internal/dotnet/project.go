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

type projectXml struct {
	XMLName        xml.Name           `xml:"Project"`
	PropertyGroups []propertyGroupXml `xml:"PropertyGroup"`
	ItemGroups     []itemGroupXml     `xml:"ItemGroup"`
}

type itemGroupXml struct {
	PackageReference []packageReferenceXml `xml:"PackageReference"`
	ProjectReference []projectReferenceXml `xml:"ProjectReference"`
}

type packageReferenceXml struct {
	Include     string `xml:"Include,attr"`
	Version     string `xml:"Version,attr"`
	VersionElem string `xml:"Version"`
}

type projectReferenceXml struct {
	Include string `xml:"Include,attr"`
}

type propertyGroupXml struct{ TargetFramework, OutputType, AssemblyName string }

type Project struct {
	RootPath          string
	Path              string
	ProjectReferences []Project
	PackageReferences []PackageReference
}

func (p Project) GetFileName() string { return filepath.Base(p.Path) }
func (p Project) GetName() string     { return strings.TrimSuffix(p.GetFileName(), ".csproj") }
func (p Project) GetRelativePath() string {
	result := p.Path
	if strings.HasPrefix(result, p.RootPath) {
		result = result[len(p.RootPath):]
	}
	result = strings.TrimPrefix(result, "/")
	return result
}
func (p Project) GetDirectoryRelativePath() string {
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
func (p Project) GetAllProjectReferences() []Project {
	var result []Project
	seen := map[string]bool{}
	result = append(result, p)
	seen[p.Path] = true
	for _, pr := range p.ProjectReferences {
		if len(pr.ProjectReferences) > 0 {
			for _, ref := range pr.GetAllProjectReferences() {
				if !seen[ref.Path] {
					result = append(result, ref)
					seen[ref.Path] = true
				}
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}
func (p Project) GetProjectReferences() []Project { return p.ProjectReferences }

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
	file, err := os.Open(path)
	if err != nil {
		if !isMain {
			slog.Warn("Cannot open file. Skipped", "path", path, "err", err)
			return Project{}, errMissingProject
		}
		return Project{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return Project{}, err
	}
	var px projectXml
	if err = xml.Unmarshal(data, &px); err != nil {
		return Project{}, err
	}
	projectReferences := util.SelectMany(util.Where(px.ItemGroups,
		func(ig itemGroupXml) bool { return len(ig.ProjectReference) > 0 }),
		func(ig itemGroupXml) []projectReferenceXml { return ig.ProjectReference })
	var references []Project
	for _, pr := range projectReferences {
		childPath := strings.Replace(pr.Include, "\\", "/", -1)
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
		func(ig itemGroupXml) bool { return len(ig.PackageReference) > 0 }),
		func(ig itemGroupXml) []packageReferenceXml { return ig.PackageReference })
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

func LoadProject(path, rootPath string) (Project, error) {
	return innerLoadProject(path, true, rootPath, []string{})
}
