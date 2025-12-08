package dotnet

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/n2jsoft-public-org/dockerfile-generator/internal/common"
	"github.com/n2jsoft-public-org/dockerfile-generator/internal/config"
	"github.com/n2jsoft-public-org/dockerfile-generator/internal/generator"
)

// TemplateContext is the data model used to render the dotnet Dockerfile template.
type TemplateContext struct {
	AdditionalFilePaths   []common.AdditionalFilePath
	Project               Project
	Config                config.Config
	BaseImage             string
	BaseSdkImage          string
	SdkVersion            string
	ApplicationEntrypoint string
}

// DotnetGenerator implements generator.Generator for .NET projects.
//
//revive:disable-next-line:exported
type DotnetGenerator struct{}

// Name returns the canonical language key for this generator.
func (d DotnetGenerator) Name() string { return config.LanguageDotnet }

// Detect returns true if the provided path looks like a single .csproj or a directory containing exactly one .csproj.
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

// Load resolves the target project (or the single .csproj inside the directory) and returns the project graph + additional context files.
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
		slog.Debug("resolved single project file in directory", "dir", projectPath, "file", p)
	}
	if !strings.HasSuffix(strings.ToLower(p), ".csproj") {
		return nil, nil, errors.New("path must be a .csproj file for dotnet")
	}
	slog.Debug("loading dotnet project", "path", p, "repoRoot", repoRoot)
	proj, err := LoadProject(p, repoRoot)
	if err != nil {
		return nil, nil, err
	}
	references := proj.GetAllProjectReferences()
	slog.Debug("project graph loaded", "root", proj.Path, "projects", len(references))
	additional, err := LoadProjectContextFromProject(proj, repoRoot)
	if err != nil {
		return nil, nil, err
	}
	slog.Debug("additional context files discovered", "count", len(additional))
	return proj, additional, nil
}

// GenerateDockerfile renders the Dockerfile into dest using the discovered project + configuration.
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

	sdkVersion := "9.0"
	if cfg.Dotnet.SdkVersion != "" {
		sdkVersion = cfg.Dotnet.SdkVersion
	}
	applicationEntrypoint := proj.GetName() + ".dll"
	if cfg.Dotnet.ApplicationEntrypoint != "" {
		applicationEntrypoint = cfg.Dotnet.ApplicationEntrypoint
	}
	slog.Debug("dotnet image selection", "runtime", baseImage, "sdk", baseSdkImage, "sdkVersion", sdkVersion, "applicationEntrypoint", applicationEntrypoint, "additionalFiles", len(additional))

	tmpl, err := template.New("dotnet-dockerfile").Parse(defaultTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(dest) // #nosec G304 - destination path intentionally created in working directory
	if err != nil {
		return err
	}
	// Execute then close explicitly to surface close errors (errcheck compliance).
	execErr := tmpl.Execute(f, TemplateContext{
		AdditionalFilePaths:   additional,
		Project:               proj,
		Config:                cfg,
		BaseImage:             baseImage,
		BaseSdkImage:          baseSdkImage,
		SdkVersion:            sdkVersion,
		ApplicationEntrypoint: applicationEntrypoint,
	})
	closeErr := f.Close()
	if execErr != nil {
		return execErr
	}
	return closeErr
}

func init() { generator.Register(DotnetGenerator{}) }
