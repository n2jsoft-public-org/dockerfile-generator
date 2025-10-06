// Package golang implements Dockerfile generation logic for Go projects.
package golang

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/common"
	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/config"
	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/generator"
)

//go:embed dockerfile.tmpl
var goTemplate string

// GoProject describes a Go module root (directory containing go.mod).
type GoProject struct {
	RootPath string
	Path     string // directory containing go.mod
	Name     string
}

// goTemplateContext is the template data for Go Dockerfile generation.
type goTemplateContext struct {
	Project         GoProject
	Config          config.Config
	BuildImage      string
	RuntimeImage    string
	BuildPackages   []string
	RuntimePackages []string
}

// GoGenerator implements generator.Generator for Go projects.
type GoGenerator struct{}

// Name returns the language key for Go.
func (g GoGenerator) Name() string { return config.LanguageGo }

// Detect returns true if path is a directory containing go.mod or a go.mod file.
func (g GoGenerator) Detect(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, nil
	}
	if info.IsDir() {
		_, err = os.Stat(filepath.Join(path, "go.mod"))
		if err == nil {
			slog.Debug("go project detected (directory contains go.mod)", "path", path)
			return true, nil
		}
		return false, nil
	}
	if strings.HasSuffix(path, "go.mod") {
		slog.Debug("go project detected (go.mod file)", "path", path)
		return true, nil
	}
	return false, nil
}

// Load gathers basic module information and returns a GoProject.
func (g GoGenerator) Load(projectPath, repoRoot string) (generator.ProjectData, []common.AdditionalFilePath, error) {
	p := projectPath
	info, err := os.Stat(p)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() && !strings.HasSuffix(p, "go.mod") {
		return nil, nil, fmt.Errorf("path must be a directory containing go.mod or the go.mod file itself")
	}
	if !info.IsDir() {
		p = filepath.Dir(p)
	}
	modPath := filepath.Join(p, "go.mod")
	// #nosec G304 - path is derived from user-provided value and constrained to directory + 'go.mod'.
	modData, err := os.ReadFile(modPath)
	if err != nil {
		return nil, nil, err
	}
	slog.Debug("reading go module file", "path", modPath, "bytes", len(modData))
	name := filepath.Base(p)
	for _, l := range strings.Split(string(modData), "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "module ") {
			parsed := filepath.Base(strings.TrimSpace(strings.TrimPrefix(l, "module ")))
			name = parsed
			slog.Debug("parsed module name", "module", parsed)
		}
	}
	proj := GoProject{RootPath: repoRoot, Path: p, Name: name}
	slog.Debug("go project loaded", "module", name, "path", p)
	return proj, nil, nil
}

// GenerateDockerfile produces a Dockerfile at dest for the given project.
func (g GoGenerator) GenerateDockerfile(
	project generator.ProjectData,
	additional []common.AdditionalFilePath,
	dest string,
	cfg config.Config) error {
	proj, ok := project.(GoProject)
	if !ok {
		return fmt.Errorf("invalid project type for go generator")
	}
	buildImage := "golang:${GO_VERSION}-alpine"
	runtimeImage := "alpine:3.19"
	if cfg.BaseBuild.Image != "" {
		buildImage = cfg.BaseBuild.Image
	}
	if cfg.Base.Image != "" {
		runtimeImage = cfg.Base.Image
	}
	slog.Debug("go image selection", "build", buildImage, "runtime", runtimeImage, "additionalFiles", len(additional))
	ctx := goTemplateContext{
		Project: proj, Config: cfg, BuildImage: buildImage, RuntimeImage: runtimeImage,
		BuildPackages: cfg.BaseBuild.Packages, RuntimePackages: cfg.Base.Packages,
	}
	tmpl, err := template.New("go-dockerfile").Parse(goTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(dest) // #nosec G304 - controlled output path
	if err != nil {
		return err
	}
	execErr := tmpl.Execute(f, ctx)
	closeErr := f.Close()
	if execErr != nil {
		return execErr
	}
	return closeErr
}

func init() { generator.Register(GoGenerator{}) }
