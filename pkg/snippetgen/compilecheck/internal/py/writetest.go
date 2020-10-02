package py

import (
	"bufio"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/compilecheck/internal/langutil"
)

// writeTest generates the test file used to check generated snippets for client libraries.
func writeTest(ctx *checkContext) (err error) {
	ctx.MethodSorted = make([]langutil.MethodID, 0, len(ctx.MethodInits))
	for m := range ctx.MethodInits {
		ctx.MethodSorted = append(ctx.MethodSorted, m)
	}
	sort.Sort(langutil.MethodSlice(ctx.MethodSorted))

	out, err := ctx.fs.Create(filepath.Join(ctx.tstDir, checkFileName))
	if err != nil {
		return err
	}
	defer func() {
		if err2 := out.Close(); err == nil {
			err = err2
		}
	}()

	bw := bufio.NewWriter(out)
	defer func() {
		if err2 := bw.Flush(); err == nil {
			err = err2
		}
	}()

	return testTmpl.Execute(bw, ctx)
}

var testTmpl = template.Must(template.New("test").Funcs(template.FuncMap{
	"testName": func(id langutil.MethodID) string {
		return strings.Replace("test_"+id.APIName+"_"+id.APIVersion+"_"+id.FragmentName, ".", "_", -1)
	},
	"lines": func(s string) []string {
		return strings.Split(s, "\n")
	},
	"pyTypes": func(s string) []string {
		switch s {
		case "boolean":
			return []string{"bool"}
		case "integer":
			return []string{"int", "long"}
		case "number":
			return []string{"int", "long", "float"}
		case "object":
			return []string{"dict"}
		case "string":
			return []string{"basestring"}
		case "list":
			return []string{"list"}
		}
		return []string{}
	},
}).Option("missingkey=error").Parse(`
from __future__ import print_function
import keyword
import sys

tests = []
top_errors = 0

{{range .MethodSorted -}}
{{$test_name := testName . -}}
def {{$test_name}}():
  global top_errors
{{- range $i, $line := lines (index $.MethodInits .)}}
{{if $line}}  {{$line}}{{end}}
{{- end}}

{{- range $param := (index $.MethodParamSets .)}}
  if keyword.iskeyword('{{$param.Name}}') or '{{$param.Name}}' in dir(__builtins__):
    if not (
{{- range $j, $type := pyTypes .Type -}}
{{if ne $j 0}} or {{end -}}
{{if eq $type "long" -}}sys.version_info[0] < 3 and {{end}}isinstance({{$param.Name}}_, {{$type}})
{{- end -}}
):
      print("In {{$test_name}}, expected {{.Name}}_ to be {{.Type}}, found %s" % type({{.Name}}_))
      top_errors += 1
  else:
    if not (
{{- range $j, $type := pyTypes .Type -}}
{{if ne $j 0}} or {{end -}}
{{if eq $type "long" -}}sys.version_info[0] < 3 and {{end}}isinstance({{$param.Name}}, {{$type}})
{{- end -}}
):
      print("In {{$test_name}}, expected {{.Name}} to be {{.Type}}, found %s" % type({{.Name}}))
      top_errors += 1
{{- end}}

tests.append({{$test_name}})

{{end -}}
for test in tests:
  test()
if top_errors > 0:
  print("%s errors" % top_errors)
  sys.exit(1)
`))
