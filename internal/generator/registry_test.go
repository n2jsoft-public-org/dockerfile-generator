package generator

import (
	"testing"

	"github.com/n2jsoft-public-org/dockerfile-generator/internal/common"
	"github.com/n2jsoft-public-org/dockerfile-generator/internal/config"
)

type mockGen struct{ name string }

func (m mockGen) Name() string                { return m.name }
func (m mockGen) Detect(string) (bool, error) { return false, nil }
func (m mockGen) Load(string, string) (ProjectData, []common.AdditionalFilePath, error) {
	return nil, nil, nil
}
func (m mockGen) GenerateDockerfile(ProjectData, []common.AdditionalFilePath, string, config.Config) error {
	return nil
}

func mustPanic(t *testing.T, fn func(), msg string) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic: %s", msg)
		}
	}()
	fn()
}

func TestRegistryBasic(t *testing.T) {
	origReg := registry
	origOrdered := ordered
	registry = map[string]Generator{}
	ordered = nil
	defer func() { registry = origReg; ordered = origOrdered }()

	mustPanic(t, func() { Register(nil) }, "nil generator should panic")
	mustPanic(t, func() { Register(mockGen{name: ""}) }, "empty name should panic")

	g1 := mockGen{name: "lang1"}
	Register(g1)
	if len(All()) != 1 {
		t.Fatalf("expected 1 generator, got %d", len(All()))
	}
	if got, ok := Get("lang1"); !ok || got.Name() != "lang1" {
		t.Fatalf("Get failed: %#v %v", got, ok)
	}
	if MustGet("lang1").Name() != "lang1" {
		t.Fatalf("MustGet returned wrong generator")
	}

	g2 := mockGen{name: "lang1"}
	Register(g2)
	if len(All()) != 1 {
		t.Fatalf("duplicate register should not increase ordered slice")
	}
	if got, _ := Get("lang1"); got != g2 {
		t.Fatalf("registry not updated to new instance")
	}

	mustPanic(t, func() { MustGet("missing") }, "MustGet missing should panic")
}
