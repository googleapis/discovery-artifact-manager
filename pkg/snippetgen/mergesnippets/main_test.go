package main

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/fragment"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/metadata"
)

func TestMerge(t *testing.T) {
	// Set this to true to preserve the actual and expected output
	// on disk after this test runs.
	preserveOutput := true

	tempDir, err := ioutil.TempDir("", "merge_test_")
	if err != nil {
		t.Fatalf("could not create tempdir")
	}
	if !preserveOutput {
		defer func(path string) {
			if err := os.RemoveAll(path); err != nil {
				t.Fatalf("could not RemoveAll(%q)", path)
			}
		}(tempDir)
	}

	*primaryLocation = path.Join(tempDir, "primary")
	*secondaryLocation = path.Join(tempDir, "secondary")
	*mergedLocation = path.Join(tempDir, "merged")

	apiName := "harvest"
	apiVersion := "v12"
	apiMethod := "harvest.field.read"

	primaryFragment := fragment.Info{
		Path: fragment.Path{
			APIName:         apiName,
			APIVersion:      apiVersion,
			SnippetRevision: "371",
			FragmentName:    apiMethod,
			Lang:            metadata.FragmentLanguage,
		},
		File: fragment.File{
			Format:      "0.314",
			APIName:     apiName,
			APIVersion:  apiVersion,
			APIRevision: "371",
			ID:          apiMethod,
			CodeFragment: map[string]*fragment.CodeFragment{
				"Java": {
					GenerationVersion: "gv11",
					GenerationDate:    "today",
					Fragment:          "// A Java Sample",
					Libraries: []*fragment.LibraryInfo{
						{
							URL:  "foo.com",
							Name: "The FooJava client library",
						},
					},
				},
			},
		},
	}

	secondaryFragment := fragment.Info{
		Path: fragment.Path{
			APIName:         apiName,
			APIVersion:      apiVersion,
			SnippetRevision: "7",
			FragmentName:    apiMethod,
			Lang:            metadata.FragmentLanguage,
		},
		File: fragment.File{
			Format:      "0.314",
			APIName:     apiName,
			APIVersion:  apiVersion,
			APIRevision: "7",
			ID:          apiMethod,
			CodeFragment: map[string]*fragment.CodeFragment{
				"Python": {
					GenerationVersion: "89",
					GenerationDate:    "last week",
					Fragment:          "# A Python sample",
					Libraries: []*fragment.LibraryInfo{
						{
							URL:  "bar.edu",
							Name: "The BarPython client library",
						},
					},
				},
			},
		},
	}

	mergedFragment := fragment.Info{
		Path: fragment.Path{
			APIName:         apiName,
			APIVersion:      apiVersion,
			SnippetRevision: metadata.TimestampShort,
			FragmentName:    apiMethod,
			Lang:            metadata.FragmentLanguage,
		},
		File: fragment.File{
			Format:      "0.314",
			APIName:     apiName,
			APIVersion:  apiVersion,
			APIRevision: "371~7",
			ID:          apiMethod,
			CodeFragment: map[string]*fragment.CodeFragment{
				"Python": {
					GenerationVersion: "89",
					GenerationDate:    "last week",
					Fragment:          "# A Python sample",
					Libraries: []*fragment.LibraryInfo{
						{
							URL:  "bar.edu",
							Name: "The BarPython client library",
						},
					},
				},
				"Java": {
					GenerationVersion: "1[gv11(today)]+[()]",
					GenerationDate:    metadata.Timestamp,
					Fragment:          "// A Java Sample",
					Libraries: []*fragment.LibraryInfo{
						{
							URL:  "foo.com",
							Name: "The FooJava client library",
						},
					},
				},
			},
		},
	}

	if err := primaryFragment.ToFile(*primaryLocation, false); err != nil {
		t.Fatalf("could not write to primary location %q: %s\nFragment:\n%#v", *primaryLocation, err, primaryFragment)
	}

	if err := secondaryFragment.ToFile(*secondaryLocation, false); err != nil {
		t.Fatalf("could not write to secondary location %q: %s\nFragment:\n%#v", *secondaryLocation, err, secondaryFragment)
	}

	if preserveOutput {
		mergedWant := path.Join(tempDir, "merged-want")
		if err := mergedFragment.ToFile(mergedWant, false); err != nil {
			t.Fatalf("could not write to merged-want location %q: %s\nFragment:\n%#v", *mergedLocation, err, mergedFragment)
		}
	}

	main()

	// Check with the actual revision number.
	actualMergedFilename := path.Join(*mergedLocation, apiName, apiVersion, metadata.TimestampShort, apiMethod+".frag.json")
	actualMerged, err := fragment.FromFile(actualMergedFilename)
	if err != nil {
		t.Errorf("could not load merged fragment %q", actualMergedFilename)
	} else if !reflect.DeepEqual(mergedFragment.File, actualMerged.File) {
		t.Errorf("merge error:\ngot:\n%#v\nwant:\n%#v", actualMerged, mergedFragment)
	}

	// Check with the "current" revision number.
	actualMergedFilename = path.Join(*mergedLocation, apiName, apiVersion, metadata.CurrentRevision, apiMethod+".frag.json")
	actualMerged, err = fragment.FromFile(actualMergedFilename)
	if err != nil {
		t.Errorf("could not load merged fragment %q", actualMergedFilename)
	} else if !reflect.DeepEqual(mergedFragment.File, actualMerged.File) {
		t.Errorf("merge error:\ngot:\n%#v\nwant:\n%#v", *actualMerged, mergedFragment)
	}
}
