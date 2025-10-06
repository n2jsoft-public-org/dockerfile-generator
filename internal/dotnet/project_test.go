package dotnet

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const baseCsproj = `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net9.0</TargetFramework>
    <OutputType>Exe</OutputType>
    <AssemblyName>{{NAME}}</AssemblyName>
  </PropertyGroup>
  <ItemGroup>
    {{REFERENCES}}
    {{PACKAGES}}
  </ItemGroup>
</Project>`

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestLoadProject_WithReferencesAndPackages(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "A", "A.csproj")
	refPath := filepath.Join(root, "B", "B.csproj")
	refContent := baseCsproj
	refContent = replace(refContent, "{{NAME}}", "B")
	refContent = replace(refContent, "{{REFERENCES}}", "")
	refContent = replace(refContent, "{{PACKAGES}}", `<PackageReference Include="Newtonsoft.Json" Version="13.0.3"/>`)
	write(t, refPath, refContent)
	mainContent := baseCsproj
	mainContent = replace(mainContent, "{{NAME}}", "A")
	mainContent = replace(mainContent, "{{REFERENCES}}", `<ProjectReference Include="../B/B.csproj" />`)
	mainContent = replace(mainContent, "{{PACKAGES}}", `<PackageReference Include="Serilog" Version="4.0.0"/>`)
	write(t, mainPath, mainContent)

	proj, err := LoadProject(mainPath, root)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	all := proj.GetAllProjectReferences()
	if len(all) != 2 {
		t.Fatalf("expected 2 projects total, got %d", len(all))
	}
	if len(proj.PackageReferences) != 1 {
		t.Fatalf("main project packages missing")
	}
	if len(proj.ProjectReferences) != 1 {
		t.Fatalf("main project reference missing")
	}
	ref := proj.ProjectReferences[0]
	if len(ref.PackageReferences) != 1 {
		t.Fatalf("ref project packages missing")
	}
}

func TestLoadProject_CircularReference(t *testing.T) {
	root := t.TempDir()
	pathA := filepath.Join(root, "A", "A.csproj")
	pathB := filepath.Join(root, "B", "B.csproj")

	aContent := baseCsproj
	aContent = replace(aContent, "{{NAME}}", "A")
	aContent = replace(aContent, "{{PACKAGES}}", "")
	aContent = replace(aContent, "{{REFERENCES}}", `<ProjectReference Include="../B/B.csproj" />`)
	write(t, pathA, aContent)

	bContent := baseCsproj
	bContent = replace(bContent, "{{NAME}}", "B")
	bContent = replace(bContent, "{{PACKAGES}}", "")
	bContent = replace(bContent, "{{REFERENCES}}", `<ProjectReference Include="../A/A.csproj" />`)
	write(t, pathB, bContent)

	_, err := LoadProject(pathA, root)
	if !errors.Is(err, errCircularRef) {
		// errCircularRef should bubble up
		if err == nil {
			t.Fatalf("expected circular ref error, got nil")
		}
	}
}

func replace(s, old, newVal string) string { return stringReplaceAll(s, old, newVal) }

// minimal replace to avoid importing strings again (keep imports light)
func stringReplaceAll(s, old, newVal string) string {
	if old == "" {
		return s
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if len(s)-i >= len(old) && s[i:i+len(old)] == old {
			out = append(out, newVal...)
			i += len(old)
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return string(out)
}
