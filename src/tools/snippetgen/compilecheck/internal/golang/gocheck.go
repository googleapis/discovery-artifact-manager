// Package golang implements compile checking for Go code samples.
//
// The name slightly deviates from the norm: while the Java compile checking is in the package
// "java", we cannot name this package "go", since it is a reserved keyword.
package golang

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gapi-cmds/src/snippetgen/compilecheck/internal/filesys"
)

// Check sets up the Go compile check, satisfying checker.Func. It copies each snipppet into its
// own package changing the package name from "main" to "sample". Separating the packages prevents
// names in the snnipets from colliding, and renaming the packages prevents "go build" from
// linking all the files.
func Check(files []string, libDir, tstDir string) (string, error) {
	libGopath := filepath.Join(libDir, "gopath")
	if err := os.RemoveAll(tstDir); err != nil {
		return "", err
	}
	if err := setupGopath(libGopath); err != nil {
		return "", err
	}

	for i, fname := range files {
		dst := filepath.Join(tstDir, fmt.Sprintf("p%d", i), "sample.go")
		if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
			return "", err
		}
		if err := copyFile(fname, dst, filesys.OS{}); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf(
		"# Make sure Go samples work\n(export GOPATH=%s; go get google.golang.org/api/...; cd %s; go build ./...)",
		libGopath, tstDir), nil
}

// setupGopath sets up a GOPATH directory so we don't collide with the user's.
func setupGopath(gopath string) error {
	dirs := []string{"src", "bin", "pkg"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(gopath, d), 0755); err != nil {
			return err
		}
	}
	return nil
}

// copyFile copies a Go source file from `srcFile` to `{dstDir}/sample.go`.
func copyFile(srcFile, dstFile string, fs filesys.FS) error {
	src, err := fs.ReadFile(srcFile)
	if err != nil {
		return err
	}
	src = bytes.Replace(src, []byte("package main"), []byte("package sample"), -1)

	return fs.WriteFile(dstFile, src, 0644)
}
