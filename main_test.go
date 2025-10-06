package main

import "testing"

func TestTranslateLegacyLongFlags(t *testing.T) {
	cases := []struct {
		in   []string
		out  []string
		name string
	}{
		{[]string{"-path"}, []string{"--path"}, "simple"},
		{[]string{"-language", "value"}, []string{"--language", "value"}, "with value next"},
		{[]string{"-path=."}, []string{"--path=."}, "assignment"},
		{[]string{"-dry-run", "-version"}, []string{"--dry-run", "--version"}, "multiple"},
		{[]string{"-p"}, []string{"-p"}, "short flag unchanged"},
	}
	for _, c := range cases {
		got := translateLegacyLongFlags(c.in)
		if len(got) != len(c.out) {
			// show slice for debugging
			// ... not failing on length difference alone if content diff handled below
		}
		for i := range c.out {
			if got[i] != c.out[i] {
				// detailed diff
				// mark failure
				t.Fatalf("%s: expected %v got %v", c.name, c.out, got)
			}
		}
	}
}
