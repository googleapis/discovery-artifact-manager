package checker

import (
	"bytes"
	"fmt"

	"discovery-artifact-manager/snippetgen/compilecheck/internal/csharp"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/golang"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/java"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/js"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/nodejs"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/php"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/py"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/ruby"
)

// check abstracts setting up compile checks for code samples.
//
// Files to be checked should be listed in `files`.
// Client libraries we test against should be stored in `libDir`.
// Artifacts we create to set up the tests should be stored in `tstDir`.
// Further instructions to run tests are returned, along with any error.
type check func(files []string, libDir, tstDir string) (string, error)

// initFn abstracts the language-specific, API-independent
// initialization. If `force` is set, the initialization always
// occurs. Otherwise, the initialization may be skipped if it can be
// determined that the environment looks like it's already set up as
// needed.
type initFn func(force bool) (string, error)

// Checkers maps language file extension both to the check function
// implementing test-setup for that language and to the initFn
// implementing the language-specific set-up for that language.
var Checkers = map[string]struct {
	Fn   check
	Init initFn
}{
	"go":   {Fn: golang.Check},
	"java": {Fn: java.Check},
	"js":   {Fn: js.Check},
	"njs":  {Fn: nodejs.Check},
	"php":  {Fn: php.Check},
	"rb":   {Fn: ruby.Check},
	"cs":   {Fn: csharp.Check, Init: csharp.Init},
	"py":   {Fn: py.Check},
}

// Init runs the API-independent initialization code for all
// languages. It returns the concatenated output and the first error
// encountered.
func Init(force bool) (string, error) {
	output := &bytes.Buffer{}
	for lang, c := range Checkers {
		if c.Init != nil {
			out, err := c.Init(force)
			output.Write([]byte(out))
			if err != nil {
				return output.String(), fmt.Errorf("error initializing %q: %s\n", lang, err)
			}
		}
	}
	return output.String(), nil
}
