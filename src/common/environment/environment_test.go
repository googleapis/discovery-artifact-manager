package environment

import (
	"path/filepath"
	"testing"
)

func TestRepoRoot(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatalf("unexpected error: %q", err)
	}

	if got, want := filepath.Base(root), "discovery-artifact-manager"; got != want {
		t.Errorf("last path element (%q): got: %q want %q", root, got, want)
	}
}
