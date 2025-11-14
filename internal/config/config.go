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
	// DefaultLanguage retained for backward compatibility (no longer auto-applied unless config file present).
	DefaultLanguage = LanguageDotnet
)

// Config represents the top-level configuration.
type Config struct {
	Language  string       `yaml:"language"`
	Dotnet    DotnetConfig `yaml:"dotnet"`
	Base      ImageConfig  `yaml:"base"`
	BaseBuild ImageConfig  `yaml:"base-build"`
	Final     FinalConfig  `yaml:"final"`
}

// DotnetConfig represents .NET-specific configuration.
type DotnetConfig struct {
	SdkVersion string `yaml:"sdk-version"`
}

// FinalConfig represents configuration applied to the final runtime image.
type FinalConfig struct {
	Run []string `yaml:"run"`
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

// Default returns a Config with no language preset so autodetection can occur
// if the user does not provide a config file or explicit flag.
func Default() Config {
	return Config{}
}
