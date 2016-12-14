package nodejs

import (
	"bufio"
	"path/filepath"
	"sort"
	"text/template"

	"discovery-artifact-manager/tools/snippetgen/compilecheck/internal/filesys"
	"discovery-artifact-manager/tools/snippetgen/compilecheck/internal/langutil"
)

// renderCheck outputs the JavaScript code used for compile-checking the generated code samples.
// The code is placed under `dir` using `creator`. Returns the name of the file written and any
// error encountered.
func renderCheck(dir string, creator filesys.Creator, methodInits langutil.MethodInitializers, libMethods langutil.MethodParamSets) (string, error) {
	arg := makeArg(methodInits, libMethods)
	fname := filepath.Join(dir, "check.js")
	wr, err := creator.Create(fname)
	if err != nil {
		return fname, err
	}
	bw := bufio.NewWriter(wr)

	err = testTemplate.Execute(bw, arg)
	if err2 := bw.Flush(); err == nil {
		err = err2
	}
	if err2 := wr.Close(); err == nil {
		err = err2
	}
	return fname, err
}

type templateArg struct {
	IDs         []langutil.MethodID
	MethodInits map[langutil.MethodID]string
	LibMethods  map[langutil.MethodID][]langutil.MethodParam
}

// makeArg prepares the argument to be passed into the template. In particular, it orders the
// methods so that the template renders deterministically.
func makeArg(methodInits langutil.MethodInitializers, libMethods langutil.MethodParamSets) templateArg {
	ids := make(langutil.MethodSlice, 0, len(methodInits))
	for id := range methodInits {
		ids = append(ids, id)
	}
	sort.Sort(ids)
	return templateArg{
		IDs:         ids,
		MethodInits: methodInits,
		LibMethods:  libMethods,
	}
}

var testTemplate = template.Must(template.New("test").Parse(`
{{range $method := .IDs}}
// {{$method.APIName}}/{{$method.APIVersion}}/{{$method.FragmentName}}
(function() {
var typ;
var authClient = {};
var apiKey = '';
{{index $.MethodInits $method}}
{{range $param := index $.LibMethods $method}}
typ = typeof(request.{{$param.Name}});
if (typ !== {{printf "%q" $param.Type}}) {
  console.log({{printf "%s/%s/%s: typeof(request.%s), want '%s' got '" $method.APIName $method.APIVersion $method.FragmentName $param.Name $param.Type | printf "%q"}} + typ + "'");
}
{{end}}
})();
{{end}}
`)).Option("missingkey=error")
