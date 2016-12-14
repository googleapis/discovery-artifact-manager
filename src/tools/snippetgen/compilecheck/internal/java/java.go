// Package java implements compilecheck for Java.
package java

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gapi-cmds/src/snippetgen/common/clientlib"
	"gapi-cmds/src/snippetgen/common/fragment"
)

// Check sets up the Java compile check, satisfying compilecheck.checker.
func Check(files []string, libDir, tstDir string) (string, error) {
	// Remove old srcs to make space
	if err := os.RemoveAll(tstDir); err != nil {
		return "", err
	}
	for _, dir := range []string{libDir, tstDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return "", err
		}
	}

	clientLibs, err := requiredLibraries(files)
	if err != nil {
		return "", err
	}

	if err := clientlib.DownloadUnzipIfMissing(clientLibs, libDir); err != nil {
		return "", err
	}

	dstFiles, err := copyClasses(files, tstDir)
	if err != nil {
		return "", err
	}

	jarFiles, err := listJars(libDir)
	if err != nil {
		return "", err
	}

	javacOptPath := filepath.Join(tstDir, "javacopt")
	if err := ioutil.WriteFile(javacOptPath, generateJavacOpt(jarFiles, dstFiles), 0640); err != nil {
		return "", err
	}
	return fmt.Sprintf("# Make sure java compilation works\njavac @%s", javacOptPath), nil
}

// requiredLibs parses file names in `fnames` and returns a list of client libraries needed to
// check those files.
func requiredLibraries(fnames []string) ([]clientlib.Lib, error) {
	type libID struct {
		Name, Version string
	}
	libSet := make(map[libID]bool)
	for _, fname := range fnames {
		p, err := fragment.ParseFileName(fname)
		if err != nil {
			return nil, err
		}
		libSet[libID{
			Name:    p.APIName,
			Version: p.APIVersion,
		}] = true
	}

	libs := make([]clientlib.Lib, 0, len(libSet))
	for l := range libSet {
		url, err := clientlib.DownloadURL("Java", l.Name, l.Version)
		if err != nil {
			return nil, err
		}
		libs = append(libs, clientlib.Lib{
			Name: l.Name,
			URL:  url,
		})
	}
	return libs, nil
}

// classNameRegexp is used to search for class names in Java files.
//
// Code fragment files are typically named myservice.v1.1234.mymethod.frag.java.
// However, javac expects the class Foo to be in the file Foo.java, so we have to know the class
// name of the code fragment.
var classNameRegexp = regexp.MustCompile(`class\s+(\w+)\s+\{`)

// copyClasses copies java source code from `files`,
// into Java files, each with a unique package name PKG under dstDir/PKG/ClassName.java.
func copyClasses(files []string, dstDir string) ([]string, error) {
	nDir := 0
	var dstFiles []string
	for _, fname := range files {
		nDir++
		pkgName := fmt.Sprintf("p%d", nDir)
		dstFile, err := copyClass(fname, dstDir, pkgName)
		if err != nil {
			return nil, err
		}
		dstFiles = append(dstFiles, dstFile)
	}
	return dstFiles, nil
}

// copyClass copies a java source code from file `srcFile`, gives it a package `pkg`, and then
// writes the content into `dstDir/pkg/ClassName.java`.
func copyClass(srcFile, dstDir, pkg string) (string, error) {
	content, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return "", err
	}

	subs := classNameRegexp.FindSubmatch(content)
	if len(subs) == 0 {
		return "", fmt.Errorf("cannot find class name: %s", srcFile)
	}

	pkgDir := filepath.Join(dstDir, pkg)
	dstFile := string(subs[1]) + ".java"

	if err := os.MkdirAll(pkgDir, 0750); err != nil {
		return "", err
	}

	var dstContent bytes.Buffer
	fmt.Fprintf(&dstContent, "package %s;\n", pkg)
	dstContent.Write(content)

	dst := filepath.Join(pkgDir, dstFile)

	return dst, ioutil.WriteFile(dst, dstContent.Bytes(), 0640)
}

// listJars lists all .jar files recursively under directory root.
// It does not follow symlinks.
func listJars(root string) ([]string, error) {
	var jarFiles []string
	err := filepath.Walk(root, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".jar") {
			jarFiles = append(jarFiles, path)
		}
		return nil
	})
	return jarFiles, err
}

// javacOpt prepares javac options with which the user can run javac to perform the tests
func generateJavacOpt(jarFiles, testFiles []string) []byte {
	var javacOpt bytes.Buffer
	fmt.Fprintf(&javacOpt, "-cp '%s'", strings.Join(jarFiles, ":"))
	for _, f := range testFiles {
		javacOpt.WriteString(" ")
		javacOpt.WriteString(f)
	}
	return javacOpt.Bytes()
}
