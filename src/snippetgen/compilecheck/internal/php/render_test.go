package php

import (
	"io/ioutil"
	"testing"

	"discovery-artifact-manager/snippetgen/compilecheck/internal/filesys"
)

var (
	testDir = "/tmp/snippetgen/test/php/"
)

func TestRender(t *testing.T) {
	opener := filesys.MapFS{}
	sample := Sample{
		Service:   "Sample_Service",
		InitLines: []string{"$param = '';"},
		MethodSignature: MethodSignature{
			Identifier: "SampleID",
			Path:       "Sample_Service->lib",
			Method:     "sampleMethod",
			Params:     []string{"param"},
		},
	}
	parsedLib := ParsedLib{
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "SampleClass",
				MethodName:    "sampleMethod",
				ParameterName: "paramA",
			},
			Types: []string{"typeA"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "SampleClass",
				MethodName:    "sampleMethod",
				ParameterName: "paramB",
			},
			Types: []string{"typeB", "typeC"},
		},
	}

	mainFilePath, parsedLibFilePath, err := render(testDir, opener, []Sample{sample}, parsedLib)
	if err != nil {
		t.Fatal(err)
	}

	actualMain := opener[mainFilePath]
	expectedMain, err2 := ioutil.ReadFile("testdata/template.baseline")
	if err2 != nil {
		t.Fatal(err2)
	}
	if actualMain != string(expectedMain) {
		t.Errorf("Main PHP file baseline test failed.\n Expected:\n%s Actual:\n%s",
			expectedMain, actualMain)
	}

	actualParsedLib := opener[parsedLibFilePath]
	expectedParsedLib, err3 := ioutil.ReadFile("testdata/parsedlib_template.baseline")
	if err3 != nil {
		t.Fatal(err3)
	}
	if actualParsedLib != string(expectedParsedLib) {
		t.Errorf("Parsedlib PHP file baseline test failed.\n Expected:\n%s Actual:\n%s",
			expectedParsedLib, actualParsedLib)
	}
}
