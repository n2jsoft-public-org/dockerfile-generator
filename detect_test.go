// main is the package declaration for the entry point of the Go application.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/config"
	_ "github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/dotnet"
	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/generator"
	_ "github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/golang"
)

const sampleCsproj = `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net9.0</TargetFramework>
    <OutputType>Exe</OutputType>
    <AssemblyName>App</AssemblyName>
  </PropertyGroup>
</Project>`

const sampleGoMod = `module example.com/app

go 1.23`

func writeFile(t *testing.T, path, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestRegistryHasGenerators(t *testing.T) {
	if len(generator.All()) == 0 {
		t.Fatalf("expected at least one registered generator")
	}
	if _, ok := generator.Get("dotnet"); !ok {
		t.Fatalf("dotnet generator missing")
	}
	if _, ok := generator.Get("go"); !ok {
		t.Fatalf("go generator missing")
	}
}

func TestDotnetDetectAndGenerate(t *testing.T) {
	dir := t.TempDir()
	projPath := filepath.Join(dir, "App.csproj")
	writeFile(t, projPath, sampleCsproj)
	gen, ok := generator.Get("dotnet")
	if !ok {
		t.Fatalf("dotnet generator not found")
	}
	okDetect, err := gen.Detect(dir)
	if err != nil || !okDetect {
		t.Fatalf("detect failed: %v", err)
	}
	root := dir // treat temp dir as root (simulate .git) by creating a .git folder
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatalf("git dir: %v", err)
	}
	proj, additional, err := gen.Load(projPath, root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(additional) != 0 {
		t.Fatalf("expected no additional files, got %d", len(additional))
	}
	dest := filepath.Join(dir, "Dockerfile")
	cfg := config.Default()
	if err := gen.GenerateDockerfile(proj, additional, dest, cfg); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, err := os.ReadFile(dest) // #nosec G304 - reading generated file in test
	if err != nil {
		t.Fatalf("read dockerfile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "dotnet --no-restore publish") {
		t.Fatalf("dockerfile missing dotnet publish step")
	}
}

func TestGoDetectAndGenerate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), sampleGoMod)
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}")
	gen, ok := generator.Get("go")
	if !ok {
		t.Fatalf("go generator not found")
	}
	okDetect, err := gen.Detect(dir)
	if err != nil || !okDetect {
		t.Fatalf("detect failed: %v", err)
	}
	root := dir
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatalf("git dir: %v", err)
	}
	proj, additional, err := gen.Load(dir, root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(additional) != 0 {
		t.Fatalf("expected no additional files, got %d", len(additional))
	}
	dest := filepath.Join(dir, "Dockerfile")
	cfg := config.Default()
	if err := gen.GenerateDockerfile(proj, additional, dest, cfg); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, err := os.ReadFile(dest) // #nosec G304 - reading generated file in test
	if err != nil {
		t.Fatalf("read dockerfile: %v", err)
	}
	if !strings.Contains(string(data), "GO_VERSION") {
		t.Fatalf("dockerfile missing GO_VERSION ARG")
	}
}
