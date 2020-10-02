// Package nodejs implements compilecheck for NodeJS.
//
// The snippet generation process derives its information from the discovery doc that should work
// with the client library. We perform a "compile check" by extracting the type information from
// the client library documentation, and making sure they match the types we deduced.
//
// NOTE: Object types in JavaScript are simply called "object", so it is not possible to verify that
// an object parameter has the correct "type". However, if the documentation also references members
// of the object, we will verify that the member has the appropriate type.
package nodejs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/discovery-artifact-manager/pkg/common/errorlist"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/clientlib"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/fragment"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/compilecheck/internal/filesys"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/compilecheck/internal/langutil"
)

var (
	// clientLibAPIRoot is the subdirectory where we should look for API libraries.
	// It is written in array form so that they may be portably joined with system-dependent separator.
	clientLibAPIRoot = []string{"nodejs-client", "google-api-nodejs-client-master", "apis"}
)

// Check sets up test environment for NodeJS.
func Check(files []string, libDir, tstDir string) (string, error) {
	if err := os.RemoveAll(tstDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(tstDir, 0750); err != nil {
		return "", err
	}
	url, err := clientlib.DownloadURL("Node.js", "", "")
	if err != nil {
		return "", err
	}
	clientLib := []clientlib.Lib{clientlib.Lib{"nodejs-client", url}}
	if err := clientlib.DownloadUnzipIfMissing(clientLib, libDir); err != nil {
		return "", err
	}

	methodInits, err := readFiles(files, filesys.OS{})
	if err != nil {
		return "", err
	}

	libRoot := filepath.Join(libDir, filepath.Join(clientLibAPIRoot...))
	libMethods, err := parseLibs(methodInits, libRoot, filesys.OS{})
	if err != nil {
		return "", err
	}

	fname, err := renderCheck(tstDir, filesys.OS{}, methodInits, libMethods)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("# Make sure Node samples work\nnode %s", fname), nil
}

// readFiles reads code samples from files in `fname` and returns a map of MethodID to that method's
// initialization code. The content of each file is read using `opener`.
func readFiles(fnames []string, opener filesys.Opener) (langutil.MethodInitializers, error) {
	var errlist errorlist.Errors
	files := make(langutil.MethodInitializers)
	for _, fname := range fnames {
		if id, initText, err := readFile(fname, opener); err != nil {
			errlist.Add(err)
		} else {
			files[id] = initText
		}
	}
	return files, errlist.Error()
}

var (
	startInit = []byte("var request = {")
	endInit   = []byte("};")
)

// readFile reads a single code sample file `fname` using `opener`, returning
// MethodID describing the method the sample is for,
// the code the sample uses to initialize its RPC request,
// and any error encountered.
func readFile(fname string, opener filesys.Opener) (langutil.MethodID, string, error) {
	path, err := fragment.ParseFileName(fname)
	if err != nil {
		return langutil.MethodID{}, "", err
	}
	id := langutil.MethodID{
		APIName:      path.APIName,
		APIVersion:   path.APIVersion,
		FragmentName: path.FragmentName,
	}

	file, err := opener.Open(fname)
	if err != nil {
		return langutil.MethodID{}, "", err
	}
	defer file.Close()

	var initText bytes.Buffer
	var inInit bool
	sc := bufio.NewScanner(file)

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if !inInit && bytes.Equal(line, startInit) {
			inInit = true
		}
		if inInit {
			initText.Write(line)
			initText.WriteRune('\n')
		}
		if inInit && bytes.Equal(line, endInit) {
			inInit = false
		}
	}
	return id, initText.String(), sc.Err()
}
