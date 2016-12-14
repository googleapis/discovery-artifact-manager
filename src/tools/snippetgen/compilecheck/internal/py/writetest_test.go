package py

import (
	"testing"

	"gapi-cmds/src/snippetgen/compilecheck/internal/filesys"
	"gapi-cmds/src/snippetgen/compilecheck/internal/langutil"

	"github.com/pmezard/go-difflib/difflib"
)

func TestWriteTest(t *testing.T) {
	methodID := langutil.MethodID{APIName: "pubsub", APIVersion: "v1", FragmentName: "projects.subscriptions.create"}
	mapFS := filesys.MapFS{}

	ctx := checkContext{
		tstDir: "path/to/tst",
		fs:     mapFS,
		MethodInits: langutil.MethodInitializers{
			methodID: `
# TODO: Change placeholders below to appropriate parameter values for the 'create' method:
# * The name of the subscription. It must have the format ` + "`" + `"projects/{project}/subscriptions/{subscription}"` + "`" + `. ` + "`" + `{subscription}` + "`" + ` must start with a letter, and contain only letters (` + "`" + `[A-Za-z]` + "`" + `), numbers (` + "`" + `[0-9]` + "`" + `), dashes (` + "`" + `-` + "`" + `), underscores (` + "`" + `_` + "`" + `), periods (` + "`" + `.` + "`" + `), tildes (` + "`" + `~` + "`" + `), plus (` + "`" + `+` + "`" + `) or percent signs (` + "`" + `%` + "`" + `). It must be between 3 and 255 characters in length, and it must not start with ` + "`" + `"goog"` + "`" + `.
name = 'projects/{MY-PROJECT}/subscriptions/{MY-SUBSCRIPTION}'

body = {
# TODO: Add desired entries of the 'body' dict
}

`,
		},
		MethodParamSets: langutil.MethodParamSets{
			methodID: {
				{Name: "name", Type: "string"},
				{Name: "body", Type: "object"},
			},
		},
	}

	if err := writeTest(&ctx); err != nil {
		t.Fatal(err)
	}

	got := mapFS["path/to/tst/check.py"]
	want := `
from __future__ import print_function
import sys

tests = []
top_errors = 0

def test_pubsub_v1_projects_subscriptions_create():
  global top_errors

  # TODO: Change placeholders below to appropriate parameter values for the 'create' method:
  # * The name of the subscription. It must have the format ` + "`" + `"projects/{project}/subscriptions/{subscription}"` + "`" + `. ` + "`" + `{subscription}` + "`" + ` must start with a letter, and contain only letters (` + "`" + `[A-Za-z]` + "`" + `), numbers (` + "`" + `[0-9]` + "`" + `), dashes (` + "`" + `-` + "`" + `), underscores (` + "`" + `_` + "`" + `), periods (` + "`" + `.` + "`" + `), tildes (` + "`" + `~` + "`" + `), plus (` + "`" + `+` + "`" + `) or percent signs (` + "`" + `%` + "`" + `). It must be between 3 and 255 characters in length, and it must not start with ` + "`" + `"goog"` + "`" + `.
  name = 'projects/{MY-PROJECT}/subscriptions/{MY-SUBSCRIPTION}'

  body = {
  # TODO: Add desired entries of the 'body' dict
  }


  if not (isinstance(name, basestring)):
    print("In test_pubsub_v1_projects_subscriptions_create, expected name to be string, found %s" % type(name))
    top_errors += 1
  if not (isinstance(body, dict)):
    print("In test_pubsub_v1_projects_subscriptions_create, expected body to be object, found %s" % type(body))
    top_errors += 1

tests.append(test_pubsub_v1_projects_subscriptions_create)

for test in tests:
  test()
if top_errors > 0:
  print("%s errors" % top_errors)
  sys.exit(1)
`

	if got != want {
		diff := difflib.ContextDiff{
			A:        difflib.SplitLines(got),
			B:        difflib.SplitLines(want),
			FromFile: "got",
			ToFile:   "want",
			Context:  3,
			Eol:      "\n",
		}
		result, err := difflib.GetContextDiffString(diff)
		if err != nil {
			t.Errorf("wrong test program: got:\n%q\nwant:\n%q", got, want)
		} else {
			t.Errorf("wrong test program: got:\n%q\nwant:\n%q\ndiff:\n%v", got, want, result)
		}
	}
}
