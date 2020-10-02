package main

import (
	"go/format"
	"testing"

	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/fragment"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/metadata"
)

func TestFormatterNames(t *testing.T) {
	langNames := make(map[string]bool)
	for _, l := range metadata.AllowedLanguages {
		langNames[l.Name] = true
	}
	for _, fmter := range formatters {
		if name := fmter.langName; !langNames[name] {
			t.Errorf("Formatter exists for language %q but the language is not defined", name)
		}
	}
}

func TestGoFormat(t *testing.T) {
	t.Parallel()

	// Valid program that needs to be reformatted for sure
	goContent := `
package foo

func Bar() {
	var x= 123 * 4+5}
`

	want, err := format.Source([]byte(goContent))
	if err != nil {
		t.Fatal(err)
	}
	if string(want) == goContent {
		t.Fatalf("the test program must not already be properly formatted")
	}

	path := fragment.Path{
		APIName:         "service",
		APIVersion:      "v1",
		SnippetRevision: "1234",
		FragmentName:    "method",
		Lang:            metadata.FragmentLanguage,
	}
	frags := fragmentLanguageMap{
		path: map[string]*fragment.CodeFragment{
			"Go": {
				Fragment: goContent,
			},
		},
	}
	if err := formatFragments(frags); err != nil {
		t.Fatal(err)
	}
	if got := frags[path]["Go"].Fragment; got != string(want) {
		t.Errorf("wrong format, got\n%q\n, want\n%q", got, want)
	}
}

func TestJavaFormat(t *testing.T) {
	t.Parallel()

	content := "public  class     Foo{}"
	want := "public class Foo {}\n"

	path := fragment.Path{
		APIName:         "service",
		APIVersion:      "v1",
		SnippetRevision: "1234",
		FragmentName:    "method",
		Lang:            metadata.FragmentLanguage,
	}
	frags := fragmentLanguageMap{
		path: map[string]*fragment.CodeFragment{
			"Java": {
				Fragment: content,
			},
		},
	}
	if err := formatFragments(frags); err != nil {
		t.Fatal(err)
	}
	if got := frags[path]["Java"].Fragment; got != want {
		t.Errorf("wrong format, got %q, want %q", got, want)
	}
}
