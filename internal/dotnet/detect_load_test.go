package dotnet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n2jsoft-public-org/dockerfile-generator/internal/config"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestDotnetGeneratorDetect(t *testing.T) {
	g := DotnetGenerator{}
	tdir := t.TempDir()
	file := filepath.Join(tdir, "App.csproj")
	writeFile(t, file, `<Project/>`)
	if ok, _ := g.Detect(file); !ok {
		t.Fatalf("expected detect true on direct file")
	}
	if ok, _ := g.Detect(tdir); !ok {
		t.Fatalf("expected detect true on dir with single csproj")
	}
	emptyDir := t.TempDir()
	if ok, _ := g.Detect(emptyDir); ok {
		t.Fatalf("expected detect false on empty directory")
	}
	multiDir := t.TempDir()
	writeFile(t, filepath.Join(multiDir, "A.csproj"), `<Project/>`)
	writeFile(t, filepath.Join(multiDir, "B.csproj"), `<Project/>`)
	if ok, _ := g.Detect(multiDir); ok {
		t.Fatalf("expected detect false on multi directory")
	}
}

func TestDotnetGeneratorLoadErrors(t *testing.T) {
	g := DotnetGenerator{}
	empty := t.TempDir()
	if _, _, err := g.Load(empty, empty); err == nil {
		t.Fatalf("expected error for empty directory")
	}
	// directory with multiple csproj
	multi := t.TempDir()
	writeFile(t, filepath.Join(multi, "A.csproj"), `<Project/>`)
	writeFile(t, filepath.Join(multi, "B.csproj"), `<Project/>`)
	if _, _, err := g.Load(multi, multi); err == nil {
		t.Fatalf("expected error for multiple csproj")
	}
	// non csproj file
	non := filepath.Join(t.TempDir(), "file.txt")
	writeFile(t, non, "hello")
	if _, _, err := g.Load(non, filepath.Dir(non)); err == nil {
		t.Fatalf("expected error for non csproj file")
	}
}

func TestProjectReferencesAndPackages(t *testing.T) {
	root := t.TempDir()
	// child1 project with package references (attribute & element forms)
	child1 := filepath.Join(root, "Child1.csproj")
	writeFile(t, child1,
		`<Project><ItemGroup><PackageReference Include="PkgA" Version="1.0"/><PackageReference Include="PkgB"><Version>2.0</Version></PackageReference></ItemGroup></Project>`)
	// child2 referenced but missing -> should be skipped silently

	// root project referencing both
	rootFile := filepath.Join(root, "Root.csproj")
	writeFile(t, rootFile,
		`<Project><ItemGroup><ProjectReference Include="Child1.csproj"/><ProjectReference Include="Child2.csproj"/></ItemGroup></Project>`)
	proj, err := LoadProject(rootFile, root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	refs := proj.GetProjectReferences()
	if len(refs) != 1 || refs[0].GetFileName() != "Child1.csproj" {
		t.Fatalf("expected single child1 ref, got %+v", refs)
	}
	if len(refs[0].PackageReferences) != 2 {
		t.Fatalf("expected 2 package refs, got %d", len(refs[0].PackageReferences))
	}
	// ensure GetProjectReferences (direct) vs GetAllProjectReferences difference
	all := proj.GetAllProjectReferences()
	if len(all) != 2 {
		t.Fatalf("expected 2 total projects (root + child1), got %d", len(all))
	}
}

func TestCircularProjectReference(t *testing.T) {
	root := t.TempDir()
	projA := filepath.Join(root, "A.csproj")
	projB := filepath.Join(root, "B.csproj")
	writeFile(t, projA, `<Project><ItemGroup><ProjectReference Include="B.csproj"/></ItemGroup></Project>`)
	writeFile(t, projB, `<Project><ItemGroup><ProjectReference Include="A.csproj"/></ItemGroup></Project>`)
	_, err := LoadProject(projA, root)
	if err == nil {
		t.Fatalf("expected circular reference error")
	}
}

func TestGenerateDockerfileInvalidProjectType(t *testing.T) {
	g := DotnetGenerator{}
	dir := t.TempDir()
	dest := filepath.Join(dir, "Dockerfile")
	err := g.GenerateDockerfile(struct{}{}, nil, dest, config.Default())
	if err == nil {
		t.Fatalf("expected error for invalid project type")
	}
}
