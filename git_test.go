package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasGitDirAndFindRepositoryRoot(t *testing.T) {
	tdir := t.TempDir()
	gitDir := filepath.Join(tdir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	deep := filepath.Join(tdir, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}

	if !hasGitDir(tdir) {
		t.Fatalf("expected hasGitDir true at root")
	}
	if hasGitDir(deep) {
		t.Fatalf("expected hasGitDir false in nested path")
	}

	root := findRepositoryRoot(deep)
	if root != tdir {
		t.Fatalf("expected repo root %s got %s", tdir, root)
	}

	// path without any git root -> empty string
	plain := t.TempDir()
	if r := findRepositoryRoot(plain); r != "" {
		t.Fatalf("expected empty root, got %q", r)
	}
}

func TestIsRootPath(t *testing.T) {
	if !isRootPath("/") {
		t.Fatalf("/ should be root path")
	}
	tdir := t.TempDir()
	if isRootPath(tdir) {
		t.Fatalf("temp dir should not be root")
	}
}
