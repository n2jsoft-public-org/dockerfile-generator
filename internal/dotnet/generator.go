package dotnet

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/common"
	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/config"
	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/generator"
)

type TemplateContext struct {
	AdditionalFilePaths []common.AdditionalFilePath
	Project             Project
	Config              config.Config
	BaseImage           string
	BaseSdkImage        string
}

type DotnetGenerator struct{}

func (d DotnetGenerator) Name() string { return config.LanguageDotnet }

func (d DotnetGenerator) Detect(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, nil
	}
	if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".csproj") {
		return true, nil
	}
	if info.IsDir() {
		entries, _ := os.ReadDir(path)
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".csproj") {
				count++
			}
		}
		if count == 1 {
			return true, nil
		}
	}
	return false, nil
}

func (d DotnetGenerator) Load(projectPath, repoRoot string) (
	generator.ProjectData,
	[]common.AdditionalFilePath,
	error) {
	p := projectPath
	info, err := os.Stat(p)
	if err != nil {
		return nil, nil, err
	}
	if info.IsDir() {
		// find single csproj
		matches, _ := filepath.Glob(filepath.Join(p, "*.csproj"))
		if len(matches) == 0 {
			return nil, nil, fmt.Errorf("no .csproj found in directory %s", p)
		}
		if len(matches) > 1 {
			return nil, nil, fmt.Errorf("multiple .csproj found; specify one explicitly")
		}
		p = matches[0]
	}
	if !strings.HasSuffix(strings.ToLower(p), ".csproj") {
		return nil, nil, errors.New("path must be a .csproj file for dotnet")
	}
	proj, err := LoadProject(p, repoRoot)
	if err != nil {
		return nil, nil, err
	}
	additional, err := LoadProjectContextFromProject(proj, repoRoot)
	if err != nil {
		return nil, nil, err
	}
	return proj, additional, nil
}

func (d DotnetGenerator) GenerateDockerfile(
	project generator.ProjectData,
	additional []common.AdditionalFilePath,
	dest string,
	cfg config.Config) error {
	proj, ok := project.(Project)
	if !ok {
		return fmt.Errorf("invalid project type for dotnet generator")
	}

	var baseImage, baseSdkImage string
	if cfg.Base.Image == "" {
		baseImage = "mcr.microsoft.com/dotnet/aspnet:${TARGET_DOTNET_VERSION}-alpine"
	} else {
		baseImage = cfg.Base.Image
	}
	if cfg.BaseBuild.Image == "" {
		baseSdkImage = "mcr.microsoft.com/dotnet/sdk:${TARGET_DOTNET_VERSION}-alpine"
	} else {
		baseSdkImage = cfg.BaseBuild.Image
	}

	// Choose template (could allow override later)
	tmpl, err := template.New("dotnet-dockerfile").Parse(defaultTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, TemplateContext{
		AdditionalFilePaths: additional,
		Project:             proj,
		Config:              cfg,
		BaseImage:           baseImage,
		BaseSdkImage:        baseSdkImage,
	})
}

func init() { generator.Register(DotnetGenerator{}) }
