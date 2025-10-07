package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, DefaultDockerBuildFileName)
	content := "language: go\nbase:\n  image: alpine:3.20\n  packages:\n    - ca-certificates\nbase-build:\n  image: golang:1.23-alpine\n  packages:\n    - build-base\n"
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(file)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Language != "go" {
		t.Fatalf("expected language go got %s", cfg.Language)
	}
	if cfg.Base.Image != "alpine:3.20" || len(cfg.Base.Packages) != 1 || cfg.Base.Packages[0] != "ca-certificates" {
		t.Fatalf("base image/packages mismatch: %+v", cfg.Base)
	}
	if cfg.BaseBuild.Image != "golang:1.23-alpine" || len(cfg.BaseBuild.Packages) != 1 || cfg.BaseBuild.Packages[0] != "build-base" {
		t.Fatalf("base-build mismatch: %+v", cfg.BaseBuild)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Language != "" { // language should now be empty to allow autodetect
		t.Fatalf("expected default language empty got %s", cfg.Language)
	}
}
