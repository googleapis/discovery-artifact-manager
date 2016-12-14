package ruby

import (
	"bufio"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gapi-cmds/src/snippetgen/compilecheck/internal/langutil"
)

func writeTest(ctx *checkContext) error {
	ctx.MethodSorted = make([]langutil.MethodID, 0, len(ctx.MethodInits))
	for m := range ctx.MethodInits {
		ctx.MethodSorted = append(ctx.MethodSorted, m)
	}
	sort.Sort(langutil.MethodSlice(ctx.MethodSorted))

	out, err := ctx.fs.Create(filepath.Join(ctx.tstDir, checkFileName))
	if err != nil {
		return err
	}

	bw := bufio.NewWriter(out)
	err = testTmpl.Execute(bw, ctx)
	if err2 := bw.Flush(); err == nil {
		err = err2
	}
	if err2 := out.Close(); err == nil {
		err = err2
	}
	return err
}

var testTmpl = template.Must(template.New("test").Funcs(template.FuncMap{
	"splitComma": func(s string) []string {
		ar := strings.Split(s, ",")
		for i, v := range ar {
			ar[i] = strings.TrimSpace(v)
		}
		return ar
	},
}).Option("missingkey=error").Parse(`
{{/*
ifclause is a template used for checking type.
It evalutates to one of two forms:
  not(foo.is_a? Bar)
and
  not(!!foo == foo)
The latter is used to check if "foo" is a boolean, as Ruby doesn't have a Boolean type.
If there are multiple types, separated by commas, the not expressions are logical-and'ed together.
*/}}
{{- define "ifclause" -}}
{{range $i, $t := splitComma .Type -}}
{{if ne $i 0}} && {{end -}}
not({{if eq $t "Boolean"}}!!{{$.Name}} == {{$.Name}}{{else}}{{$.Name}}.is_a? {{$t}}{{end}})
{{- end -}}
{{end -}}

tests = []
top_errors = 0

{{range .MethodSorted}}
tests << Proc.new {
{{index $.MethodInits .}}

{{range $param := (index $.MethodParamSets .)}}
  if {{template "ifclause" .}}
    puts "Expected {{.Name}} to be {{.Type}}, found #{ {{.Name}}.class.name }"
    top_errors += 1
  end
{{end}}
}
{{end}}

for t in tests do
  t.call
end
if top_errors != 0 then
  puts "#{top_errors} errors"
	exit 1
end
`))
