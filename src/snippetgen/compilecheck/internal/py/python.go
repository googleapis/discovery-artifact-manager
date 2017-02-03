// Package py implements compilecheck for Python.
//
// The snippet generation process derives its information from the Discovery doc that should work
// with the client library. We perform a "compile check" by extracting the type information from
// the client library documentation, and making sure it matches the types we deduced.
package py

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"discovery-artifact-manager/common/parsehtml"
	"discovery-artifact-manager/snippetgen/common/clientlib"
	"discovery-artifact-manager/snippetgen/common/fragment"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/filesys"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/langutil"
)

var (
	// clientLibAPIRoot is the subdirectory where we should look for API libraries.
	clientLibAPIRoot = filepath.Join("python-client", "google-api-python-client-master", "docs", "dyn")

	// checkFileName is the name of the file we write "compile check" code to.
	checkFileName = "check.py"
)

var (
	// initPattern matches a sample file with an excerpt of the following form ([] denotes optional):
	//
	// 	service = ...
	//
	// 	[<initialization code>]
	//
	// 	[<resourcesObject> = ...]
	// 	request = ...
	initPattern = regexp.MustCompile(`\nservice =.*\n\n((?:[\s\S]*\n)?)(?:.*\n)?request =`)

	// initRequestPattern matches sample initialization code for a request object with an excerpt of the following form:
	//
	// 	<request_name> = {
	// 	    # TODO: Add desired entries of the request body.[ Only assigned entries
	// 	    # will be changed:]
	// 	}
	//
	initRequestPattern = regexp.MustCompile(
		`\w+ = {\n([\s\S]*?)    # TODO: Add desired entries to the request body\.( Only assigned entries\n    # will be changed\.| All existing entries\n    # will be replaced\.)?\n}`)

	// initRequestGeneric uses the generic name 'body' for the request object (used in Python
	// client libraries) in place of the request object name in initRequestPattern
	initRequestGeneric = `body = {
$1    # TODO: Add desired entries to the request body.$2
}`
)

type checkContext struct {
	// These need to be initialized before use
	checkFiles     []string
	libDir, tstDir string
	fs             filesys.FS

	// These get populated as we go
	MethodInits     langutil.MethodInitializers
	MethodParamSets langutil.MethodParamSets
	MethodSorted    []langutil.MethodID
}

// Check sets up the test environment for Python.
func Check(files []string, libDir, tstDir string) (string, error) {
	if err := os.RemoveAll(tstDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(tstDir, 0750); err != nil {
		return "", err
	}
	url, err := clientlib.DownloadURL("Python", "", "")
	if err != nil {
		return "", err
	}
	clientLib := []clientlib.Lib{clientlib.Lib{"python-client", url}}
	if err := clientlib.DownloadUnzipIfMissing(clientLib, libDir); err != nil {
		return "", err
	}

	ctx := checkContext{
		checkFiles: files,
		libDir:     libDir,
		tstDir:     tstDir,
		fs:         filesys.OS{},
	}

	fns := []func(*checkContext) error{
		readMethodInits,
		parseLibs,
		writeTest,
	}
	for _, f := range fns {
		if err := f(&ctx); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("# Make sure Python samples work\npython %s", filepath.Join(tstDir, checkFileName)), nil
}

// readFiles reads code samples from files in `ctx.checkFiles` and populates `ctx.MethodInits`.
// The content of each file is read using `ctx.fs`.
func readMethodInits(ctx *checkContext) error {
	ctx.MethodInits = make(langutil.MethodInitializers)
	for _, fname := range ctx.checkFiles {
		if id, initText, err := readFile(fname, ctx.fs); err == nil {
			ctx.MethodInits[id] = initText
		} else {
			return err
		}
	}
	return nil
}

// readFile reads a single code sample file `fname` using `opener`, returning MethodID describing
// the method the sample is for, the code the sample uses to initialize its RPC request,
// and any error encountered.
func readFile(fname string, opener filesys.Opener) (langutil.MethodID, string, error) {
	path, err := fragment.ParseFileName(fname)
	if err != nil {
		return langutil.MethodID{}, "", err
	}
	id := langutil.MethodID{
		APIName:      path.APIName,
		APIVersion:   path.APIVersion,
		FragmentName: parsehtml.TrimPast(path.FragmentName, "."),
	}

	content, err := opener.ReadFile(fname)
	if err != nil {
		return id, "", err
	}
	initSection := initPattern.FindStringSubmatch(string(content))
	if initSection == nil {
		return id, "", fmt.Errorf("cannot extract parameter initialization code: %s", fname)
	}
	init := initRequestPattern.ReplaceAllString(string(initSection[1]), initRequestGeneric)
	return id, init, nil
}
