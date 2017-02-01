package snippet

import (
	"fmt"
	"testing"

	"discovery-artifact-manager/snippetgen/common/fragment"
	"discovery-artifact-manager/snippetgen/common/metadata"
)

func TestValidateMerged(t *testing.T) {
	mrg := &Merger{}
	frag := &fragment.Info{Path: fragment.Path{}, File: fragment.File{}}
	key := frag.Key()

	mrg.mergedFragments = make(fragmentMap)
	mrg.mergedFragments[key] = frag

	cases := []struct {
		languages []metadata.Language
		valid     bool
	}{
		{
			languages: nil,
			valid:     true,
		},
		{
			languages: []metadata.Language{{"foo language", "", "foo", true}},
			valid:     false,
		},
		{
			languages: metadata.RequiredLanguages,
			valid:     true,
		},
		{
			languages: append(metadata.RequiredLanguages, metadata.Language{"foo language", "", "foo", true}),
			valid:     false,
		},
		{
			languages: []metadata.Language{},
			valid:     true,
		},
		{
			languages: []metadata.Language{{"Python", "", "py", true}},
			valid:     true,
		},
	}

	for idx, test := range cases {
		mrg.errorList.Clear()
		frag.File.CodeFragment = make(map[string]*fragment.CodeFragment)
		for _, lang := range test.languages {
			frag.File.CodeFragment[lang.Name] = &fragment.CodeFragment{Fragment: "sample"}
		}
		if got, want := mrg.Error() == nil, true; got != want {
			t.Errorf("%d: initial error is nil: got: %v, want: %v\n%s", idx, got, want, mrg.Error())
		}
		mrg.validateMergedFragments()
		if got, want := mrg.Error() == nil, test.valid; got != want {
			t.Errorf("%d: validateMergedFragments: got: %v, want: %v\nlanguages: %#v", idx, got, want, test.languages)
		}
	}
}

func TestRenameLanguages(t *testing.T) {
	displayLanguages := []string{
		"Java",
		"C#",
		"PHP",
		"Python",
		"Ruby",
		"Dart",
		"Go",
		"Google Web Toolkit",
		"Web",
		"Objective-C",
		"Node.js",
		"Code Fragment",
	}

	mrg := &Merger{}
	frag := &fragment.Info{Path: fragment.Path{}, File: fragment.File{}}
	key := frag.Key()

	mrg.mergedFragments = make(fragmentMap)
	mrg.mergedFragments[key] = frag
	frag.File.CodeFragment = make(map[string]*fragment.CodeFragment)

	for _, lang := range metadata.AllowedLanguages {
		frag.File.CodeFragment[lang.Name] = &fragment.CodeFragment{Fragment: fmt.Sprintf("/* sample in %q */", lang.Name)}
	}

	if got, want := mrg.Error() == nil, true; got != want {
		t.Errorf("initial error is nil: got: %v, want: %v\n%s", got, want, mrg.Error())
	}

	mrg.renameLanguages = true
	mrg.renameFragmentLanguages()

	if got, want := mrg.Error() == nil, true; got != want {
		t.Errorf("final error is nil: got: %v, want: %v\n%s", got, want, mrg.Error())
	}

	if got, want := len(frag.File.CodeFragment), len(displayLanguages); got != want {
		t.Errorf("unexpected number of languages: got: %d, want %d", got, want)
	}
	for _, d := range displayLanguages {
		if _, exist := frag.File.CodeFragment[d]; !exist {
			t.Errorf("language %q expected but not seen", d)
		}
	}

}

func TestGetLatestRevision(t *testing.T) {
	inputPaths := []string{
		"some/path/alice/v1/0",
		"some/path/alice/v1/20160629",
		"some/path/alice/v1/20150320",
		"some/path/alice/v1/20160627",
		"some/path/alice/v2/20160701",
		"some/path/bob/v1/20150320",
		"some/path/bob/v1/20150621",
		"some/path/alice/v2/rev21",
		"v1/20130211", // error
	}

	expectedPaths := map[string]bool{
		"some/path/alice/v1/20160629": true,
		"some/path/bob/v1/20150621":   true,
		"some/path/alice/v2/rev21":    true,
	}

	expectedErrors := 1

	mrg := &Merger{}
	actualPaths := mrg.getLatestRevisions(inputPaths)
	if got, want := len(actualPaths), len(expectedPaths); got != want {
		t.Errorf("unexpected number of paths returned: got %d, want %d\n%s", got, want, actualPaths)
	}
	for _, p := range actualPaths {
		if found := expectedPaths[p]; !found {
			t.Errorf("unexpected path %q", p)
		}
	}

	if got, want := len(mrg.errorList), expectedErrors; got != want {
		t.Errorf("unexpected number of errors: got %d, want %d:\n%s", got, want, mrg.Error())
	}

	if got, want := len(mrg.getLatestRevisions([]string{})), 0; got != want {
		t.Errorf("getLatestRevision on empty list has wrong length: got: %d, want %d", got, want)
	}
}
