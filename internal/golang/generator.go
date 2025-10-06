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

type GoProject struct {
	RootPath string
	Path     string // directory containing go.mod
	Name     string
}

type goTemplateContext struct {
	Project         GoProject
	Config          config.Config
	BuildImage      string
	RuntimeImage    string
	BuildPackages   []string
	RuntimePackages []string
}

type GoGenerator struct{}

func (g GoGenerator) Name() string { return config.LanguageGo }

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
	modData, err := os.ReadFile(modPath)
	if err != nil {
		return nil, nil, err
	}
	slog.Debug("reading go module file", "path", modPath, "bytes", len(modData))
	// parse module line
	name := filepath.Base(p)
	lines := strings.Split(string(modData), "\n")
	for _, l := range lines {
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
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	slog.Info("writing Dockerfile", "path", dest, "module", proj.Name)
	return tmpl.Execute(f, ctx)
}

func init() { generator.Register(GoGenerator{}) }
