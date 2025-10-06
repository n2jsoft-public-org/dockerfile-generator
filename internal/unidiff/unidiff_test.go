package unidiff

import "testing"

func TestUnified_Addition(t *testing.T) {
	oldText := "line1\n"
	newText := "line1\nline2\n"
	diff := Unified(oldText, newText, "Dockerfile")
	if !containsAll(diff, []string{"--- Dockerfile (old)", "+++ Dockerfile (new)", "+line2"}) {
		t.Fatalf("diff missing expected additions:\n%s", diff)
	}
}

func TestUnified_Deletion(t *testing.T) {
	oldText := "line1\nline2\n"
	newText := "line1\n"
	diff := Unified(oldText, newText, "Dockerfile")
	if !containsAll(diff, []string{"-line2"}) {
		t.Fatalf("diff missing deletion: %s", diff)
	}
}

func TestUnified_Modification(t *testing.T) {
	oldText := "lineA\n"
	newText := "lineB\n"
	diff := Unified(oldText, newText, "f")
	if !containsAll(diff, []string{"-lineA", "+lineB"}) {
		t.Fatalf("expected modification markers: %s", diff)
	}
}

func TestUnified_EmptyOld(t *testing.T) {
	oldText := ""
	newText := "line1\n"
	diff := Unified(oldText, newText, "f")
	if !containsAll(diff, []string{"+line1"}) {
		t.Fatalf("expected addition in empty-old case: %s", diff)
	}
}

func TestUnified_Identical(t *testing.T) {
	oldText := "same\nline\n"
	newText := oldText
	diff := Unified(oldText, newText, "f")
	// Expect no + or - markers except headers (only space-prefixed lines)
	if containsAny(diff, []string{"+same", "-same"}) {
		t.Fatalf("did not expect change markers: %s", diff)
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}
func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

// Simple substring search to avoid importing strings (keep tiny & deterministic)
func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
