// Package environment returns information about the environment in
// which this binary executes.
package environment

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// RepoRoot returns this repository's root path.
func RepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not determine the filesystem location of this repo")
	}
	srcDir, err := filepath.EvalSymlinks(fmt.Sprintf("%s/../..", filepath.Dir(file)))
	if err != nil {
		return "", fmt.Errorf("could not resolve symlinks from %q to determine location of this repo", file)
	}
	return filepath.Abs(fmt.Sprintf("%s/..", srcDir))
}
