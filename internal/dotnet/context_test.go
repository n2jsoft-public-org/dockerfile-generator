package dotnet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectContextFromProject(t *testing.T) {
	root := t.TempDir()
	// Simulate repo root marker
	_ = os.Mkdir(filepath.Join(root, ".git"), 0o750)
	// Create context files
	writeFile := func(p string) {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	writeFile(filepath.Join(root, "nuget.config"))
	writeFile(filepath.Join(root, "Directory.Build.props"))
	sub := filepath.Join(root, "src", "App")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	projPath := filepath.Join(sub, "App.csproj")
	projContent := `<?xml version="1.0"?><Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net9.0</TargetFramework></PropertyGroup></Project>`
	if err := os.WriteFile(projPath, []byte(projContent), 0o600); err != nil {
		t.Fatalf("write proj: %v", err)
	}
	proj, err := LoadProject(projPath, root)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	additional, err := LoadProjectContextFromProject(proj, root)
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	if len(additional) == 0 {
		t.Fatalf("expected context files, got 0")
	}
	foundNuget := false
	for _, a := range additional {
		if filepath.Base(a.Path) == "nuget.config" {
			foundNuget = true
		}
	}
	if !foundNuget {
		t.Fatalf("nuget.config not discovered")
	}
}
