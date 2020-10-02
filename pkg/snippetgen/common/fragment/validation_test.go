package fragment

import (
	"testing"

	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/metadata"
)

func TestCheckLanguages(t *testing.T) {
	info := &Info{}

	// Test having a required language, but then restore the
	// global state.
	defer func(saved []metadata.Language) {
		metadata.RequiredLanguages = saved
	}(metadata.RequiredLanguages)
	metadata.RequiredLanguages = []metadata.Language{{"Java", "", "java", true}}

	// No languages should fail validation.
	if got, want := info.CheckLanguages() == nil, false; got != want {
		t.Errorf("no code samples fails CheckLanguages: got: %v, want: %v", got, want)
	}

	// No languages should fail validation.
	info.File.CodeFragment = make(map[string]*CodeFragment)
	if got, want := info.CheckLanguages() == nil, false; got != want {
		t.Errorf("no code samples fails CheckLanguages: got: %v, want: %v", got, want)
	}

	// An unrecognized language should fail validation
	info.File.CodeFragment["foo language"] = &CodeFragment{Fragment: "sample"}
	if got, want := info.CheckLanguages() == nil, false; got != want {
		t.Errorf("invalid language fails CheckLanguages: got: %v, want: %v", got, want)
	}

	// Excluding exactly one of the required languages should fail validation
	for _, excludedLanguage := range metadata.RequiredLanguages {
		info.File.CodeFragment = make(map[string]*CodeFragment)
		for _, language := range metadata.RequiredLanguages {
			if language != excludedLanguage {
				info.File.CodeFragment[language.Name] = &CodeFragment{Fragment: "sample"}
			}
		}
		if got, want := info.CheckLanguages() == nil, false; got != want {
			t.Errorf("missing required language %v fails CheckLanguages: got: %v, want: %v", excludedLanguage, got, want)
		}
	}

	info.File.CodeFragment = make(map[string]*CodeFragment)
	for _, language := range metadata.RequiredLanguages {
		info.File.CodeFragment[language.Name] = &CodeFragment{Fragment: "sample"}
	}
	// Having all the required languages only should pass validation
	if got, want := info.CheckLanguages() == nil, true; got != want {
		t.Errorf("having just the required languages passes CheckLanguages: got: %v, want: %v", got, want)
	}

	// Having all the required languages and a language with a different display name should pass validation
	info.File.CodeFragment["Browser"] = &CodeFragment{Fragment: "sample"}
	info.File.CodeFragment["C#"] = &CodeFragment{Fragment: "sample"}
	if err := info.CheckLanguages(); err != nil {
		t.Errorf("having the required languages plus a language with a different display name passes CheckLanguages: got error: %q", err)
	}

	// Having all the required languages and an unknown language should fail validation
	info.File.CodeFragment["foo language"] = &CodeFragment{Fragment: "sample"}
	if got, want := info.CheckLanguages() == nil, false; got != want {
		t.Errorf("having the required languages plus a disallowed language fails CheckLanguages: got: %v, want: %v", got, want)
	}
}

func TestHasConsistentMetadata(t *testing.T) {
	cases := []struct {
		info       Info
		consistent bool
	}{
		{consistent: true},
		{info: Info{Path: Path{APIName: "alice"}, File: File{APIName: "bob"}}, consistent: false},
		{info: Info{Path: Path{APIVersion: "1"}, File: File{APIName: "2"}}, consistent: false},
		{info: Info{Path: Path{FragmentName: "foo"}, File: File{ID: "bar"}}, consistent: false},
		{info: Info{Path: Path{FragmentName: "foo"}, File: File{ID: "foo"}}, consistent: true},
		{info: Info{Path: Path{SnippetRevision: "1"}, File: File{APIRevision: "2"}}, consistent: true},

		// Various combinations of the revision identifiers
		// should always yield true, since the revision is not
		// used for consistency checking.
		{info: Info{Path: Path{SnippetRevision: "0"}, File: File{APIRevision: "0"}}, consistent: true},
		{info: Info{Path: Path{SnippetRevision: "1"}, File: File{APIRevision: "0"}}, consistent: true},
	}

	for idx, test := range cases {
		if got, want := test.info.HasConsistentMetadata() == nil, test.consistent; got != want {
			t.Errorf("%d: CheckForConsistency: got %v, want %v.\nPath: %#v\nFile: %#v\n\n", idx, got, want, test.info.Path, test.info.File)
		}
	}
}
