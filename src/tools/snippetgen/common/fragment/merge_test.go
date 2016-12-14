package fragment

import (
	"fmt"
	"reflect"
	"testing"

	"gapi-cmds/src/snippetgen/common/metadata"
)

func TestUpdateMergedMetadata(t *testing.T) {
	cases := []struct {
		primary, secondary, wantMerged *CodeFragment
		simpleMetadata                 bool
	}{
		{
			wantMerged: &CodeFragment{},
		},
		{
			primary:        &CodeFragment{GenerationVersion: "v1", GenerationDate: "d1"},
			secondary:      &CodeFragment{GenerationVersion: "v2", GenerationDate: "d2"},
			simpleMetadata: true,
			wantMerged:     &CodeFragment{GenerationVersion: "v1", GenerationDate: "d1"},
		},
		{
			primary:        nil,
			secondary:      &CodeFragment{GenerationVersion: "v2", GenerationDate: "d2"},
			simpleMetadata: true,
			wantMerged:     &CodeFragment{GenerationVersion: "v2", GenerationDate: "d2"},
		},
		{
			primary:    &CodeFragment{GenerationVersion: "v1", GenerationDate: "d1"},
			secondary:  &CodeFragment{GenerationVersion: "v2", GenerationDate: "d2"},
			wantMerged: &CodeFragment{GenerationVersion: fmt.Sprintf("%s[v1(d1)]+[v2(d2)]", currentMergeVersion), GenerationDate: metadata.Timestamp},
		},
		{
			primary:    nil,
			secondary:  &CodeFragment{GenerationVersion: "v2", GenerationDate: "d2"},
			wantMerged: &CodeFragment{GenerationVersion: fmt.Sprintf("%s[()]+[v2(d2)]", currentMergeVersion), GenerationDate: metadata.Timestamp},
		},
		{
			primary:    &CodeFragment{GenerationVersion: "v1", GenerationDate: "d1"},
			secondary:  nil,
			wantMerged: &CodeFragment{GenerationVersion: fmt.Sprintf("%s[v1(d1)]+[()]", currentMergeVersion), GenerationDate: metadata.Timestamp},
		},
	}
	for _, c := range cases {
		cf := &CodeFragment{}
		cf.updateMergedMetadata(c.primary, c.secondary, c.simpleMetadata)
		if got, want := cf, c.wantMerged; !reflect.DeepEqual(got, want) {
			t.Errorf("got: %q  want: %q for input: %#v", got, want, c)
		}
	}
}

func TestMergedAPIRevision(t *testing.T) {
	cases := []struct {
		primary, secondary string
		simpleMetadata     bool
		want               string
	}{
		{
			primary:   "12",
			secondary: "13",
			want:      "12~13",
		},
		{
			primary:        "12",
			secondary:      "13",
			simpleMetadata: true,
			want:           "12",
		},
		{
			primary:        "",
			secondary:      "13",
			simpleMetadata: true,
			want:           "13.p",
		},
		{
			primary:        "",
			secondary:      "13.p",
			simpleMetadata: true,
			want:           "13.p",
		},
	}
	for _, c := range cases {
		if got, want := mergedAPIRevision(c.primary, c.secondary, c.simpleMetadata), c.want; got != want {
			t.Errorf("got: %q  want: %q for input: %#v", got, want, c)
		}
	}
}

func TestMergeWith(t *testing.T) {
	cases := []struct {
		label                 string
		first, second, merged *Info
		simpleMetadata        bool
		error                 bool
	}{
		{
			label:  "both nil",
			first:  nil,
			second: nil,
		},
		{
			label:  "both empty",
			first:  &Info{},
			second: &Info{},
		},
		{
			label:  "just format",
			first:  &Info{File: File{Format: "1"}},
			second: &Info{File: File{Format: "1"}},
		},
		{
			label:  "first nil",
			first:  &Info{},
			second: nil,
		},
		{
			label:  "second nil",
			first:  nil,
			second: &Info{},
		},
		{
			label:  "both same format",
			first:  &Info{File: File{Format: "one"}},
			second: &Info{File: File{Format: "one"}},
		},
		{
			label:  "different formats",
			first:  &Info{Path: Path{}, File: File{Format: "one"}},
			second: &Info{Path: Path{}, File: File{Format: "two"}},
			error:  true,
		},
		{
			label:  "disparate fragments",
			first:  &Info{Path: Path{}, File: File{Format: "1", APIVersion: "v23"}},
			second: &Info{Path: Path{}, File: File{Format: "1", APIVersion: "v24"}},
			error:  true,
		},
		{
			label:  "simple merge",
			first:  &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first"}}}},
			second: &Info{File: File{Format: "1", APIRevision: "7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second"}}}},
			merged: &Info{File: File{Format: "1", APIRevision: "5~7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: metadata.Timestamp, GenerationVersion: "1[()]+[()]"}}}},
		},
		{
			label:  "merge with dates and revisions",
			first:  &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}}}},
			second: &Info{File: File{Format: "1", APIRevision: "7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged: &Info{File: File{Format: "1", APIRevision: "5~7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: metadata.Timestamp, GenerationVersion: "1[2(today)]+[3(yesterday)]"}}}},
		},
		{
			label:  "merge different languages",
			first:  &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}}}},
			second: &Info{File: File{Format: "1", APIRevision: "7", CodeFragment: map[string]*CodeFragment{"Go": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged: &Info{File: File{Format: "1", APIRevision: "5~7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: metadata.Timestamp, GenerationVersion: "1[2(today)]+[()]"}, "Go": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
		},
		{
			label:  "merge second nil",
			first:  &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}}}},
			second: nil,
			merged: &Info{File: File{Format: "1", APIRevision: "5~", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: metadata.Timestamp, GenerationVersion: "1[2(today)]+[()]"}}}},
		},
		{
			label:  "merge first nil",
			first:  nil,
			second: &Info{File: File{Format: "1", APIRevision: "7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged: &Info{File: File{Format: "1", APIRevision: "~7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: metadata.Timestamp, GenerationVersion: "1[()]+[3(yesterday)]"}}}},
		},
		{
			label:          "merge with dates and revisions, simpleMetadata",
			first:          &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}}}},
			second:         &Info{File: File{Format: "1", APIRevision: "7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged:         &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}}}},
			simpleMetadata: true,
		},
		{
			label:          "merge different languages, simpleMetadata",
			first:          &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}}}},
			second:         &Info{File: File{Format: "1", APIRevision: "7", CodeFragment: map[string]*CodeFragment{"Go": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged:         &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}, "Go": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			simpleMetadata: true,
		},
		{
			label:          "merge second nil, simpleMetadata",
			first:          &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}}}},
			second:         nil,
			merged:         &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the first", GenerationDate: "today", GenerationVersion: "2"}}}},
			simpleMetadata: true,
		},
		{
			label:          "merge first nil, simpleMetadata",
			first:          nil,
			second:         &Info{File: File{Format: "1", APIRevision: "7", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged:         &Info{File: File{Format: "1", APIRevision: "7.p", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			simpleMetadata: true,
		},
		{
			label:          "merge first nil, simpleMetadata, second already marked",
			first:          nil,
			second:         &Info{File: File{Format: "1", APIRevision: "7.p", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged:         &Info{File: File{Format: "1", APIRevision: "7.p", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			simpleMetadata: true,
		},
		{
			label:          "merge zero-length fragment in first with second",
			first:          &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "", GenerationDate: "today", GenerationVersion: "2"}}}},
			second:         &Info{File: File{Format: "1", APIRevision: "7.p", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged:         &Info{File: File{Format: "1", APIRevision: "5~7.p", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			simpleMetadata: false,
		},
		{
			label:          "merge whitespace fragment in first with second",
			first:          &Info{File: File{Format: "1", APIRevision: "5", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: " ", GenerationDate: "today", GenerationVersion: "2"}}}},
			second:         &Info{File: File{Format: "1", APIRevision: "7.p", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: "the second", GenerationDate: "yesterday", GenerationVersion: "3"}}}},
			merged:         &Info{File: File{Format: "1", APIRevision: "5~7.p", CodeFragment: map[string]*CodeFragment{"Java": {Fragment: " ", GenerationDate: metadata.Timestamp, GenerationVersion: "1[2(today)]+[3(yesterday)]"}}}},
			simpleMetadata: false,
		},
	}

	for _, test := range cases {
		merged, err := test.first.MergeWith(test.second, test.simpleMetadata)
		if got, want := (err != nil), test.error; got != want {
			t.Errorf("%q:\n error encountered: got: %v, want: %v.\n   first:  %#v\n   second: %#v\n   error:  %s\n\n", test.label, got, want, test.first, test.second, err)
		}

		if test.merged == nil {
			continue
		}
		if got, want := merged.File.APIRevision, test.merged.File.APIRevision; got != want {
			t.Errorf("%q:\n unexpected revision: got: %q, want: %q.\n   first:  %#v\n   second: %#v\n\n", test.label, got, want, test.first, test.second)
		}
		if got, want := len(merged.File.CodeFragment), len(test.merged.File.CodeFragment); got != want {
			t.Errorf("%q:\n unexpected number of code fragments: got: %d, want: %d\n   first:  %#v\n   second: %#v\n\n", test.label, got, want, *test.first, *test.second)
		}
		for tk, tv := range test.merged.File.CodeFragment {
			mv, found := merged.File.CodeFragment[tk]
			if !found {
				t.Errorf("%q:\n code fragment for %q expected but not found.\n   first:  %#v\n   second: %#v\n\n", test.label, tk, *test.first, *test.second)
				continue
			}
			areEqual := (tv.GenerationVersion == mv.GenerationVersion &&
				tv.GenerationDate == mv.GenerationDate &&
				tv.Fragment == mv.Fragment &&
				len(tv.Libraries) == len(mv.Libraries))
			if got, want := *mv, *tv; !areEqual {
				first := "(nil)"
				if test.first != nil {
					first = fmt.Sprintf("%#v", *test.first)
				}
				second := "(nil)"
				if test.second != nil {
					second = fmt.Sprintf("%#v", *test.second)
				}
				t.Errorf("%q:\n code fragment for %q:\n   got:  %#v\n   want: %#v.\n   first:  %s\n   second: %s\n\n", test.label, tk, got, want, first, second)
			}
		}
	}
}

func TestAreCommensurate(t *testing.T) {
	cases := []struct {
		first        *File
		second       *File
		commensurate bool
	}{
		{
			first:        nil,
			second:       nil,
			commensurate: false,
		},
		{
			first:        nil,
			second:       &File{},
			commensurate: false,
		},
		{
			first:        &File{},
			second:       nil,
			commensurate: false,
		},
		{
			first:        &File{},
			second:       &File{},
			commensurate: true,
		},
		{
			first: &File{
				Format:       "one",
				ID:           "some.id",
				APIName:      "random name",
				APIVersion:   "v21",
				APIRevision:  "23",
				CodeFragment: map[string]*CodeFragment{"list": {}},
			},
			second: &File{
				Format:       "one",
				ID:           "some.id",
				APIName:      "random name",
				APIVersion:   "v21",
				APIRevision:  "23",
				CodeFragment: map[string]*CodeFragment{"delete": {}},
			},
			commensurate: true,
		},
		{
			first:        &File{Format: "one"},
			second:       &File{Format: "two"},
			commensurate: false,
		},
		{
			first:        &File{ID: "some.id"},
			second:       &File{ID: "other.id"},
			commensurate: false,
		},
		{
			first:        &File{APIName: "random name"},
			second:       &File{APIName: "deliberate name"},
			commensurate: false,
		},
		{
			first:        &File{APIVersion: "v21"},
			second:       &File{APIVersion: "v210000"},
			commensurate: false,
		},
		{
			first:        &File{APIRevision: "23"},
			second:       &File{APIRevision: "230000"},
			commensurate: true,
		},
	}

	for idx, test := range cases {
		if got, want := AreCommensurate(test.first, test.second), test.commensurate; got != want {
			t.Errorf("%d: AreCommensurate( %#v,  %#v): got %v, want %v", idx, test.first, test.second, got, want)
		}
	}
}
