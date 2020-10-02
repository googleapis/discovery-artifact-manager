// Package ruby implements compilecheck for Ruby.
//
// The snippet generation process derives its information from the Discovery doc that should work
// with the client library. We perform a "compile check" by extracting the type information from
// the client library documentation, and making sure it matches the types we deduced.
package ruby

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/clientlib"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/fragment"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/compilecheck/internal/filesys"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/compilecheck/internal/langutil"
)

var (
	// clientLibAPIRoot is the subdirectory where we should look for API libraries.
	clientLibAPIRoot = filepath.Join("ruby-client", "google-api-ruby-client-master", "generated", "google", "apis")

	// checkFileName is the name of the file we write "compile check" code to.
	checkFileName = "check.rb"

	// performCallRegex matches the first line of the method call.
	// ex: "result = service.foo_bar()" or "items = service.fetch_all"
	performCallRegex = regexp.MustCompile(`\s([a-z0-9_]+ = )?[a-z0-9_]+\.([a-z0-9_]+\(.*\)|fetch_all)`)
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
	methodRename    map[langutil.MethodID]langutil.MethodID
}

// Check sets up test environment for Ruby.
func Check(files []string, libDir, tstDir string) (string, error) {
	if err := os.RemoveAll(tstDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(tstDir, 0750); err != nil {
		return "", err
	}
	url, err := clientlib.DownloadURL("Ruby", "", "")
	if err != nil {
		return "", err
	}
	clientLib := []clientlib.Lib{clientlib.Lib{"ruby-client", url}}
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
		parseNameMap,
		parseLibs,
		writeTest,
	}
	for _, f := range fns {
		if err := f(&ctx); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("# Make sure Ruby samples work\n(gem install google-api-client; ruby %s)", filepath.Join(tstDir, checkFileName)), nil
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
		FragmentName: path.FragmentName,
	}

	content, err := opener.ReadFile(fname)
	if err != nil {
		return id, "", err
	}
	contentStr := string(content)
	if p := performCallRegex.FindStringIndex(contentStr); p != nil {
		return id, contentStr[:p[0]], nil
	}
	return id, "", fmt.Errorf("cannot find start of call code: %s", fname)
}
