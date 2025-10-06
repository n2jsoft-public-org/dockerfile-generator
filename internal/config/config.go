// Package config defines the user configuration file (.dockerbuild) schema and helpers.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultDockerBuildFileName is the name of the optional project configuration file.
	DefaultDockerBuildFileName = ".dockerbuild"

	// LanguageDotnet canonical language key for .NET projects.
	LanguageDotnet = "dotnet"
	// LanguageGo canonical language key for Go projects.
	LanguageGo = "go"
	// DefaultLanguage used when none provided.
	DefaultLanguage = LanguageDotnet
)

// Config represents the top-level configuration.
type Config struct {
	Language  string      `yaml:"language"`
	Base      ImageConfig `yaml:"base"`
	BaseBuild ImageConfig `yaml:"base-build"`
}

// ImageConfig describes an image reference and optional extra packages layer.
type ImageConfig struct {
	Image    string   `yaml:"image"`
	Packages []string `yaml:"packages"`
}

// Load reads and unmarshals a configuration file from disk.
func Load(path string) (Config, error) {
	clean := filepath.Clean(path)
	// #nosec G304 - user supplied path is intentionally read; cleaned above.
	data, err := os.ReadFile(clean)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Default returns a Config with default language set.
func Default() Config {
	return Config{
		Language: DefaultLanguage,
	}
}
