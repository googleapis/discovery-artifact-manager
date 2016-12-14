package golang

import (
	"gapi-cmds/src/snippetgen/compilecheck/internal/filesys"
	"testing"
)

func TestCopyFile(t *testing.T) {
	const content = `
package main

import "foo"

func main() {
	foo.bar()
}
`
	const want = `
package sample

import "foo"

func main() {
	foo.bar()
}
`
	fs := filesys.MapFS{
		"srcfile": content,
	}
	if err := copyFile("srcfile", "dstfile", fs); err != nil {
		t.Error(err)
	}
	if got := fs["dstfile"]; got != want {
		t.Errorf("want:\n%s\ngot:\n%s", want, got)
	}
}
