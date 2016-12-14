package php

import (
	"bufio"
	"path/filepath"
	"text/template"

	"gapi-cmds/src/snippetgen/compilecheck/internal/filesys"
)

// Names of the generated PHP files used for the compile check
const (
	checkFileName     = "check.php"
	parsedLibFileName = "parsedlib.php"
)

// render outputs the executable PHP code used for compile-checking the generated samples.
// The code is placed under `testDir` using `creator`. Returns the name of the file written and any
// error encountered.
func render(testDir string, creator filesys.Creator,
	samples []Sample, parsedLib ParsedLib) (string, string, error) {
	mainFilePath := filepath.Join(testDir, checkFileName)
	if err := renderFile(mainFilePath, creator, mainTemplate, samples); err != nil {
		return "", "", err
	}
	parsedLibFilePath := filepath.Join(testDir, parsedLibFileName)
	if err := renderFile(parsedLibFilePath, creator, parsedLibTemplate, parsedLib); err != nil {
		return "", "", err
	}
	return mainFilePath, parsedLibFilePath, nil
}

// renderFile outputs the PHP code to the given path from the given template.
func renderFile(path string, creator filesys.Creator, temp *template.Template, arg interface{}) error {
	writer, err := creator.Create(path)
	if err != nil {
		return err
	}
	bufWriter := bufio.NewWriter(writer)

	err = temp.Execute(bufWriter, arg)
	if err2 := bufWriter.Flush(); err == nil {
		err = err2
	}
	if err2 := writer.Close(); err == nil {
		err = err2
	}
	return err
}
