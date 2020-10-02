package environment

import (
	"path/filepath"
	"testing"
)

// This test case needs to pass when "go test" is invoked with a
// relative path and when it is invoked with just the package name.
func TestRepoRoot(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatalf("unexpected error: %q", err)
	}

	if got, want := filepath.Base(root), "discovery-artifact-manager"; got != want {
		t.Errorf("last path element (%q): got: %q want %q", root, got, want)
	}
}
