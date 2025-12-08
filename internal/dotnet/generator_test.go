package dotnet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n2jsoft-public-org/dockerfile-generator/internal/config"
)

func TestDotnetGenerator_MultipleCsprojError(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "A.csproj"), []byte("<Project></Project>"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "B.csproj"), []byte("<Project></Project>"), 0o600)
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
		0o600); err != nil {
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
	data, _ := os.ReadFile(dest) // #nosec G304 - test reading generated file path
	content := string(data)
	if !contains(content, "mcr.microsoft.com/dotnet/aspnet:${TARGET_DOTNET_VERSION}") {
		t.Fatalf("expected default aspnet base image in output")
	}
	if !contains(content, "ARG TARGET_DOTNET_VERSION=9.0") {
		t.Fatalf("expected default TARGET_DOTNET_VERSION=9.0 in dockerfile, got: %s", content)
	}
}

func TestDotnetGenerator_ConfigOverrides(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	projPath := filepath.Join(dir, "App.csproj")
	if err := os.WriteFile(projPath,
		[]byte(`<?xml version="1.0"?><Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net9.0</TargetFramework></PropertyGroup></Project>`),
		0o600); err != nil {
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
	data, _ := os.ReadFile(dest) // #nosec G304 - test reading generated file path
	content := string(data)
	if !contains(content, "FROM customruntime:1 AS base") || !contains(content, "FROM customsdk:1 AS build") {
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

func TestDotnetGenerator_FinalRunCommands(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	projPath := filepath.Join(dir, "App.csproj")
	if err := os.WriteFile(projPath,
		[]byte(`<?xml version="1.0"?><Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net9.0</TargetFramework></PropertyGroup></Project>`),
		0o600); err != nil {
		f := err
		t.Fatalf("write: %v", f)
	}
	proj, additional, err := g.Load(projPath, dir)
	if err != nil {
		f := err
		t.Fatalf("load: %v", f)
	}
	dest := filepath.Join(dir, "Dockerfile.finalrun")
	cfg := config.Config{Final: config.FinalConfig{Run: []string{"adduser -D testuser", "echo done"}}}
	if err := g.GenerateDockerfile(proj, additional, dest, cfg); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, _ := os.ReadFile(dest) // #nosec G304 - test reading generated file path
	content := string(data)
	if !contains(content, "RUN adduser -D testuser") || !contains(content, "RUN echo done") {
		t.Fatalf("expected final run commands in dockerfile, got: %s", content)
	}
	idxRun := index(content, "RUN adduser -D testuser")
	idxEntrypoint := index(content, "ENTRYPOINT")
	if idxRun == -1 || idxEntrypoint == -1 || idxRun > idxEntrypoint {
		t.Fatalf("expected final run commands before ENTRYPOINT; indices run=%d entrypoint=%d", idxRun, idxEntrypoint)
	}
	if !contains(content, `["dotnet", "App.dll"]`) {
		t.Fatalf("expected [\"dotnet\", \"App.dll\"], got: %s", content)
	}
}

// index returns the index of sub in s or -1 if absent (avoids importing strings)
func index(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestDotnetGenerator_CustomSdkVersion(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	projPath := filepath.Join(dir, "App.csproj")
	if err := os.WriteFile(projPath,
		[]byte(`<?xml version="1.0"?><Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>`),
		0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	proj, additional, err := g.Load(projPath, dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dest := filepath.Join(dir, "Dockerfile.sdkversion")
	cfg := config.Config{
		Dotnet: config.DotnetConfig{SdkVersion: "8.0"},
	}
	if err := g.GenerateDockerfile(proj, additional, dest, cfg); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, _ := os.ReadFile(dest) // #nosec G304 - test reading generated file path
	content := string(data)
	if !contains(content, "ARG TARGET_DOTNET_VERSION=8.0") {
		t.Fatalf("expected TARGET_DOTNET_VERSION=8.0 in dockerfile, got: %s", content)
	}
}

func TestDotnetGenerator_CustomApplicationEntrypoint(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	projPath := filepath.Join(dir, "App.csproj")
	if err := os.WriteFile(projPath,
		[]byte(`<?xml version="1.0"?><Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>`),
		0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	proj, additional, err := g.Load(projPath, dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dest := filepath.Join(dir, "Dockerfile.sdkversion")
	cfg := config.Config{
		Dotnet: config.DotnetConfig{ApplicationEntrypoint: "customentrypoint.dll"},
	}
	if err := g.GenerateDockerfile(proj, additional, dest, cfg); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, _ := os.ReadFile(dest) // #nosec G304 - test reading generated file path
	content := string(data)
	if !contains(content, `["dotnet", "customentrypoint.dll"]`) {
		t.Fatalf("expected [\"dotnet\", \"customentrypoint.dll\"], got: %s", content)
	}
}
