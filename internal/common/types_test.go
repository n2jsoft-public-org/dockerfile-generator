// revive:disable:var-naming - package name 'common' is intentional for shared types used by multiple packages.
package common

import "testing"

func TestAdditionalFilePathRelativeComputations(t *testing.T) {
	p := AdditionalFilePath{Path: "/tmp/work/root/dir/file.txt", RootPath: "/tmp/work/root"}
	if got := p.GetRelativePath(); got != "dir/file.txt" {
		t.Fatalf("rel path mismatch: %s", got)
	}
	if got := p.GetDirectoryRelativePath(); got != "dir/" {
		t.Fatalf("dir rel mismatch: %s", got)
	}
}

func TestAdditionalFilePathNoRootPrefix(t *testing.T) {
	p := AdditionalFilePath{Path: "relative/path/file.txt", RootPath: "/does/not/match"}
	if got := p.GetRelativePath(); got != "relative/path/file.txt" {
		t.Fatalf("expected unchanged path got %s", got)
	}
	if got := p.GetDirectoryRelativePath(); got != "relative/path/" {
		t.Fatalf("expected dir rel got %s", got)
	}
}
