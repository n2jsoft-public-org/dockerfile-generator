package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/config"
)

func TestGoGenerator_ConfigOverrides(t *testing.T) {
	g := GoGenerator{}
	dir := t.TempDir()
	mod := "module example.com/app\n\ngo 1.23"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	proj, additional, err := g.Load(dir, dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(additional) != 0 {
		t.Fatalf("expected no additional files")
	}
	dest := filepath.Join(dir, "Dockerfile")
	cfg := config.Config{
		Base:      config.ImageConfig{Image: "alpine:3.20", Packages: []string{"ca-certificates"}},
		BaseBuild: config.ImageConfig{Image: "golang:1.24-alpine", Packages: []string{"build-base"}},
	}
	if err := g.GenerateDockerfile(proj, nil, dest, cfg); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, _ := os.ReadFile(dest) // #nosec G304 - test reading generated file
	content := string(data)
	if !strings.Contains(content, "golang:1.24-alpine") || !strings.Contains(content, "alpine:3.20") {
		t.Fatalf("expected overridden images, got: %s", content)
	}
	if !strings.Contains(content, "ca-certificates") { // simplified
		t.Fatalf("expected runtime package in Dockerfile: %s", content)
	}
	// build-base should NOT appear (current template ignores build-stage packages)
	if strings.Contains(content, "build-base") {
		t.Fatalf("did not expect build-stage package to appear in current template: %s", content)
	}
}

func TestGoGenerator_DetectFileVsDir(t *testing.T) {
	g := GoGenerator{}
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(modPath, []byte("module m\n\ngo 1.23"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	okDir, _ := g.Detect(dir)
	okFile, _ := g.Detect(modPath)
	if !okDir || !okFile {
		t.Fatalf("expected detect true for both dir and file")
	}
}
