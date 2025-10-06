package util

import "testing"

type item struct {
	id   int
	vals []int
}

func TestWhereBasic(t *testing.T) {
	in := []int{1, 2, 3, 4, 5, 6}
	got := Where(in, func(v int) bool { return v%2 == 0 })
	exp := []int{2, 4, 6}
	if len(got) != len(exp) {
		t.Fatalf("len mismatch exp %d got %d", len(exp), len(got))
	}
	for i := range exp {
		if got[i] != exp[i] {
			t.Fatalf("expected %v got %v", exp, got)
		}
	}
}

func TestWhereEmptyInput(t *testing.T) {
	var empty []string // nil slice
	res := Where(empty, func(s string) bool { return true })
	if len(res) != 0 {
		t.Fatalf("expected empty result for nil input: %#v", res)
	}
	// distinguish between nil and empty slice creation path (not required but documents behavior)
	if res != nil {
		t.Fatalf("expected nil slice result, got allocated slice: %#v", res)
	}
}

func TestWhereNoMatches(t *testing.T) {
	in := []string{"a", "bb", "ccc"}
	res := Where(in, func(s string) bool { return len(s) > 5 })
	if len(res) != 0 {
		t.Fatalf("expected no matches: %#v", res)
	}
}

func TestSelectManyBasic(t *testing.T) {
	in := []item{{1, []int{1, 2}}, {2, []int{3}}, {3, []int{}}, {4, []int{4, 5, 6}}}
	got := SelectMany(in, func(it item) []int { return it.vals })
	exp := []int{1, 2, 3, 4, 5, 6}
	if len(got) != len(exp) {
		t.Fatalf("len mismatch exp %d got %d: %v", len(exp), len(got), got)
	}
	for i := range exp {
		if got[i] != exp[i] {
			t.Fatalf("expected %v got %v", exp, got)
		}
	}
}

func TestSelectManyAllEmpty(t *testing.T) {
	in := []item{{1, nil}, {2, []int{}}, {3, nil}}
	got := SelectMany(in, func(it item) []int { return it.vals })
	if len(got) != 0 {
		t.Fatalf("expected empty slice got %v", got)
	}
}

func TestSelectManyNilInput(t *testing.T) {
	var in []item // nil slice
	got := SelectMany(in, func(it item) []int { return it.vals })
	if got != nil {
		t.Fatalf("expected nil result, got %v", got)
	}
}

func TestWhereOrderPreserved(t *testing.T) {
	in := []int{5, 4, 3, 2, 1}
	got := Where(in, func(v int) bool { return v%2 == 1 }) // 5,3,1
	exp := []int{5, 3, 1}
	if len(got) != len(exp) {
		t.Fatalf("len exp %d got %d", len(exp), len(got))
	}
	for i := range exp {
		if got[i] != exp[i] {
			t.Fatalf("order not preserved: %v", got)
		}
	}
}
