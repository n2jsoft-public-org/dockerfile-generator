package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	DefaultDockerBuildFileName = ".dockerbuild"

	LanguageDotnet  = "dotnet"
	LanguageGo      = "go"
	DefaultLanguage = LanguageDotnet
)

type Config struct {
	Language  string      `yaml:"language"`
	Base      ImageConfig `yaml:"base"`
	BaseBuild ImageConfig `yaml:"base-build"`
}

type ImageConfig struct {
	Image    string   `yaml:"image"`
	Packages []string `yaml:"packages"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Default() Config {
	return Config{
		Language: DefaultLanguage,
	}
}
