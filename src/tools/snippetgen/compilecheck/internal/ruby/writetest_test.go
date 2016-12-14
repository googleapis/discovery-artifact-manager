package ruby

import (
	"testing"

	"discovery-artifact-manager/tools/snippetgen/compilecheck/internal/filesys"
	"discovery-artifact-manager/tools/snippetgen/compilecheck/internal/langutil"
)

func TestWriteTest(t *testing.T) {
	methodID := langutil.MethodID{APIName: "myAPI", APIVersion: "v1", FragmentName: "my.method"}
	mapFS := filesys.MapFS{}

	ctx := checkContext{
		tstDir: "path/to/tst",
		fs:     mapFS,
		MethodInits: langutil.MethodInitializers{
			methodID: `
foo = ''
bar = 0
`,
		},
		MethodParamSets: langutil.MethodParamSets{
			methodID: {{Name: "foo", Type: "String"}, {Name: "bar", Type: "Fixnum"}},
		},
	}

	if err := writeTest(&ctx); err != nil {
		t.Fatal(err)
	}

	got := mapFS["path/to/tst/check.rb"]
	want := `
tests = []
top_errors = 0


tests << Proc.new {

foo = ''
bar = 0



  if not(foo.is_a? String)
    puts "Expected foo to be String, found #{ foo.class.name }"
    top_errors += 1
  end

  if not(bar.is_a? Fixnum)
    puts "Expected bar to be Fixnum, found #{ bar.class.name }"
    top_errors += 1
  end

}


for t in tests do
  t.call
end
if top_errors != 0 then
  puts "#{top_errors} errors"
	exit 1
end
`

	if got != want {
		t.Errorf("wrong test program: got:\n%q\nwant:\n%q", got, want)
	}
}
