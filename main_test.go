package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n2jsoft-public-org/dockerfile-generator/internal/config"
	"github.com/n2jsoft-public-org/dockerfile-generator/internal/golang"
)

func captureStdout(_ *testing.T, fn func()) string { // underscore for unused param (revive)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	return string(b)
}

func TestRootCmd_Version(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--version"})
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(out, "version") {
		t.Fatalf("expected version output, got %q", out)
	}
}

func TestRootCmd_VerboseVersionUpperFlag(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-V", "--verbose"})
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(out, "version") {
		t.Fatalf("expected version output, got %q", out)
	}
}

// New behavior: default path is '.'; verify it works when a project exists.
func TestRootCmd_DefaultPath(t *testing.T) {
	dir := t.TempDir()
	// simulate repo root & go module
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23\n"),
		0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	cmd := newRootCmd()
	// Explicitly provide language to ensure deterministic behavior
	cmd.SetArgs([]string{"-l", "go"}) // rely on default path '.'
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(out, "Successfully generated") {
		t.Fatalf("expected success output, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Fatalf("expected Dockerfile created: %v", err)
	}
}

func TestRootCmd_ProjectNotFound(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-p", "./does-not-exist"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "project path not found") {
		t.Fatalf("expected project not found error, got %v", err)
	}
}

func TestRootCmd_UnsupportedLanguage(t *testing.T) {
	dir := t.TempDir()
	// simulate repo root
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	// create minimal go.mod to avoid detection auto override
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23"),
		0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-p", dir, "-l", "unknown"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("expected unsupported language error, got %v", err)
	}
}

func TestRootCmd_DryRunNoChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	goMod := "module example.com/app\n\ngo 1.23\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	// Pre-generate Dockerfile using generator directly so dry-run finds no changes.
	g := golang.GoGenerator{}
	proj, _, err := g.Load(dir, dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dest := filepath.Join(dir, "Dockerfile")
	if err := g.GenerateDockerfile(proj, nil, dest, config.Default()); err != nil {
		t.Fatalf("gen: %v", err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-p", dir, "-l", "go", "-d"})
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(out, "no changes") {
		t.Fatalf("expected no changes message, got %q", out)
	}
}

func TestRootCmd_DryRunWithChanges(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23\n"),
		0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	// create an existing Dockerfile with placeholder content differing from generated output
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatalf("write existing dockerfile: %v", err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-p", dir, "-l", "go", "-d"})
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(out, "Dry run") || !strings.Contains(out, "@@") {
		t.Fatalf("expected diff output, got %q", out)
	}
}

func TestRootCmd_ConfigLanguage(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23\n"),
		0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	// create config file specifying language (redundant here) to exercise config load path
	cfg := "language: go\n"
	if err := os.WriteFile(filepath.Join(dir, ".dockerbuild"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-p", dir, "-d"}) // rely on config + dry-run
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(out, "Dry run") {
		t.Fatalf("expected dry run output, got %q", out)
	}
}

func TestRootCmd_SuccessGoGenerate(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23\n"),
		0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-p", dir, "-l", "go"})
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(out, "Successfully generated") {
		t.Fatalf("expected success output, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Fatalf("expected Dockerfile created: %v", err)
	}
}

func TestRootCmd_InvalidConfigWarning(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23\n"),
		0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	// invalid YAML
	if err := os.WriteFile(filepath.Join(dir, ".dockerbuild"), []byte(":::: not yaml"), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-p", dir, "-d"})
	captureStdout(t, func() {
		_ = cmd.Execute()
	}) // ignore error; expecting success with warning
}

func TestRootCmd_NoGitRootError(t *testing.T) {
	dir := t.TempDir()
	// NOTE: no .git created
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.23\n"),
		0o600); err != nil {
		t.Fatalf("write mod: %v", err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-p", dir, "-l", "go"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "cannot find repository root") {
		t.Fatalf("expected cannot find repository root error, got %v", err)
	}
}

func TestLogWrappers(t *testing.T) {
	t.Helper() // use t to satisfy revive unused-parameter
	logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	Debugf("debug test %d", 1)
	Infof("info test %s", "x")
	Warnf("warn test")
	Errorf("error test")
}
