package generator

import (
	"fmt"

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/common"
	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/config"
)

// ProjectData is any language-specific project representation passed to templates.
type ProjectData interface{}

// Generator describes a language-specific Dockerfile generator.
type Generator interface {
	Name() string
	Detect(path string) (bool, error)
	Load(projectPath, repoRoot string) (ProjectData, []common.AdditionalFilePath, error)
	GenerateDockerfile(
		project ProjectData,
		additional []common.AdditionalFilePath,
		dest string,
		cfg config.Config) error
}

var registry = map[string]Generator{}
var ordered []Generator

// Register adds a generator implementation to the registry.
func Register(g Generator) {
	if g == nil {
		panic("nil generator")
	}
	if g.Name() == "" {
		panic("generator name cannot be empty")
	}
	if _, exists := registry[g.Name()]; !exists {
		ordered = append(ordered, g)
	}
	registry[g.Name()] = g
}

// Get returns a generator by name and a boolean indicating if it exists.
func Get(name string) (Generator, bool) {
	g, ok := registry[name]
	return g, ok
}

// MustGet returns a generator or panics if not found (internal convenience only).
func MustGet(name string) Generator {
	g, ok := Get(name)
	if !ok {
		panic(fmt.Sprintf("generator '%s' not registered", name))
	}
	return g
}

// All returns generators in registration order.
func All() []Generator { return ordered }
