package dotnet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/config"
)

func TestDotnetGenerator_MultipleCsprojError(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "A.csproj"), []byte("<Project></Project>"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "B.csproj"), []byte("<Project></Project>"), 0o644)
	_, _, err := g.Load(dir, dir)
	if err == nil {
		t.Fatalf("expected error for multiple csproj files")
	}
}

func TestDotnetGenerator_GenerateDefaultImages(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	projPath := filepath.Join(dir, "App.csproj")
	if err := os.WriteFile(projPath,
		[]byte(`<?xml version="1.0"?><Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net9.0</TargetFramework></PropertyGroup></Project>`),
		0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	proj, additional, err := g.Load(projPath, dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dest := filepath.Join(dir, "Dockerfile")
	if err := g.GenerateDockerfile(proj, additional, dest, config.Default()); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, _ := os.ReadFile(dest)
	content := string(data)
	if !contains(content, "mcr.microsoft.com/dotnet/aspnet:${TARGET_DOTNET_VERSION}") {
		// don't assert SDK image since code may evolve
		t.Fatalf("expected default aspnet base image in output")
	}
}

func TestDotnetGenerator_ConfigOverrides(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	projPath := filepath.Join(dir, "App.csproj")
	if err := os.WriteFile(projPath,
		[]byte(`<?xml version="1.0"?><Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net9.0</TargetFramework></PropertyGroup></Project>`),
		0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	proj, additional, err := g.Load(projPath, dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dest := filepath.Join(dir, "Dockerfile.override")
	cfg := config.Config{
		Base: config.ImageConfig{Image: "customruntime:1"}, BaseBuild: config.ImageConfig{Image: "customsdk:1"},
	}
	if err := g.GenerateDockerfile(proj, additional, dest, cfg); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, _ := os.ReadFile(dest)
	content := string(data)
	if !contains(content, "FROM customruntime:1 AS base") || !contains(content, "FROM customsdk:1 AS base_build") {
		t.Fatalf("expected override images present, got: %s", content)
	}
}

// simple substring contains to avoid importing strings repeatedly
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
