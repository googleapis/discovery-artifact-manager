package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/fragment"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/metadata"
)

func TestReadFileFrom(t *testing.T) {
	t.Parallel()

	files := []struct {
		pathElems []string
		content   string
		lang      string
		url       string
		name      string
	}{
		{[]string{"service", "v1", "1234", "method.frag.java"}, "java content", "Java", "https://developers.google.com/api-client-library/java/apis/service/v1", "Java client library"},
		{[]string{"service", "v1", "1234", "method.frag.cs"}, "c# content", ".NET", "https://developers.google.com/api-client-library/dotnet/apis/service/v1", ".NET client library"},
		{[]string{"service", "v1", "1234", "method.frag.php"}, "php content", "PHP", "https://github.com/google/google-api-php-client-services", "PHP client library"},
		{[]string{"service", "v1", "1234", "method.frag.py"}, "python content", "Python", "https://github.com/google/google-api-python-client", "Python client library"},
		{[]string{"service", "v1", "1234", "method.frag.rb"}, "ruby content", "Ruby", "https://github.com/google/google-api-ruby-client", "Ruby client library"},
		{[]string{"service", "v1", "1234", "method.frag.njs"}, "node.js content", "Node.js", "https://github.com/google/google-api-nodejs-client", "Node.js client library"},
		{[]string{"service", "v1", "1234", "method.frag.go"}, "go content", "Go", "https://github.com/google/google-api-go-client", "Go client library"},
	}
	temp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(temp)

	// Create the input files. Write their names to fileNames
	var fileNames bytes.Buffer
	for _, f := range files {
		path := filepath.Join(temp, filepath.Join(f.pathElems...))
		fmt.Fprintln(&fileNames, path)
		if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatal(err)
		}
		if err = ioutil.WriteFile(path, []byte(f.content), 0640); err != nil {
			t.Fatal(err)
		}
	}

	allFrags, err := readFilesFrom(&fileNames, "")
	if err != nil {
		t.Fatal(err)
	}
	path := fragment.Path{
		APIName:         "service",
		APIVersion:      "v1",
		SnippetRevision: "1234",
		FragmentName:    "method",
		Lang:            metadata.FragmentLanguage,
	}
	frags, ok := allFrags[path]
	if !ok {
		t.Fatalf("fragment header not found")
	}
	for _, f := range files {
		frag, ok := frags[f.lang]
		switch {
		case !ok:
			t.Errorf("fragment for %q not found", f.lang)
		case frag.Fragment != f.content:
			t.Errorf("for %q, expected %q, got %q", f.lang, f.content, frag.Fragment)
		case frag.Libraries == nil:
			t.Errorf("for %q, expected non-nil libraries field but got nil", f.lang)
		case frag.Libraries[0].URL != f.url || frag.Libraries[0].Name != f.name:
			t.Errorf("for %q, expected %q:%q, but got %q:%q", f.lang, f.url, f.name, frag.Libraries[0].URL, frag.Libraries[0].Name)
		}
	}
}

func TestWriteFilesRevision(t *testing.T) {
	t.Parallel()

	path := fragment.Path{
		APIName:         "service",
		APIVersion:      "v1",
		SnippetRevision: "1234",
		FragmentName:    "method",
		Lang:            metadata.FragmentLanguage,
	}
	fragments := fragmentLanguageMap{
		path: map[string]*fragment.CodeFragment{},
	}

	temp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(temp)

	if err = writeFiles(fragments, temp); err != nil {
		t.Error(err)
	}
	if _, err = os.Stat(filepath.Join(temp, path.Filename())); err != nil {
		t.Error(err)
	}
}
