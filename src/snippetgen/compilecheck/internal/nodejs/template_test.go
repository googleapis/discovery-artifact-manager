package nodejs

import (
	"testing"

	"discovery-artifact-manager/snippetgen/compilecheck/internal/filesys"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/langutil"
)

func TestRenderCheck(t *testing.T) {
	creator := filesys.MapFS{}
	sampleMethods := map[langutil.MethodID]string{
		{"myservice", "v1", "myservice.mymethod"}: `
var request = {
  foo: bar,
  zip: zap,
  auth: authClient
};
`,
	}
	libMethods := map[langutil.MethodID][]langutil.MethodParam{
		{"myservice", "v1", "myservice.mymethod"}: {
			{"foo", "FooType"},
			{"zip", "ZipType"},
		},
	}

	want := `

// myservice/v1/myservice.mymethod
(function() {
var typ;
var authClient = {};
var apiKey = '';

var request = {
  foo: bar,
  zip: zap,
  auth: authClient
};


typ = typeof(request.foo);
if (typ !== "FooType") {
  console.log("myservice/v1/myservice.mymethod: typeof(request.foo), want 'FooType' got '" + typ + "'");
}

typ = typeof(request.zip);
if (typ !== "ZipType") {
  console.log("myservice/v1/myservice.mymethod: typeof(request.zip), want 'ZipType' got '" + typ + "'");
}

})();

`

	fname, err := renderCheck("tst", creator, sampleMethods, libMethods)
	if err != nil {
		t.Fatal(err)
	}
	if got := creator[fname]; got != want {
		t.Errorf("got\n%q,\n\nwant\n%q", got, want)
	}
}
