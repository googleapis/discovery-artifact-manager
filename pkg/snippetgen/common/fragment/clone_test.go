package fragment

import (
	"fmt"
	"reflect"
	"testing"
)

func TestInfoCloneIncludesAllFields(t *testing.T) {
	info := Info{}
	if got, want := fmt.Sprintf("%#v", *info.Clone()), `fragment.Info{Path:fragment.Path{APIName:"", APIVersion:"", SnippetRevision:"", FragmentName:"", Lang:metadata.Language{Name:"", DisplayName:"", Ext:"", Required:false}}, File:fragment.File{Format:"", APIName:"", APIVersion:"", APIRevision:"", ID:"", CodeFragment:map[string]*fragment.CodeFragment(nil)}}`; got != want {
		t.Errorf("clone did not copy all fields:\n  got:  `%s`\n  want: `%s`\n", got, want)
	}
}

func TestPathCloneIncludesAllFields(t *testing.T) {
	path := Path{}
	if got, want := fmt.Sprintf("%#v", *path.Clone()), `fragment.Path{APIName:"", APIVersion:"", SnippetRevision:"", FragmentName:"", Lang:metadata.Language{Name:"", DisplayName:"", Ext:"", Required:false}}`; got != want {
		t.Errorf("clone did not copy all fields:\n  got:  `%s`\n  want: `%s`\n", got, want)
	}
}

func TestFileCloneIncludesAllFields(t *testing.T) {
	file := File{}
	if got, want := fmt.Sprintf("%#v", *file.Clone()), `fragment.File{Format:"", APIName:"", APIVersion:"", APIRevision:"", ID:"", CodeFragment:map[string]*fragment.CodeFragment(nil)}`; got != want {
		t.Errorf("clone did not copy all fields:\n  got:  `%s`\n  want: `%s`\n", got, want)
	}
}

func TestCodeFragmentCloneIncludesAllFields(t *testing.T) {
	codeFragment := CodeFragment{}
	if got, want := fmt.Sprintf("%#v", *codeFragment.Clone()), `fragment.CodeFragment{GenerationVersion:"", GenerationDate:"", Fragment:"", Libraries:[]*fragment.LibraryInfo(nil)}`; got != want {
		t.Errorf("clone did not copy all fields:\n  got:  `%s`\n  want: `%s`\n", got, want)
	}
}

func TestCodeFragmentClonesCorrect(t *testing.T) {
	codeFragment := &CodeFragment{
		GenerationVersion: "Alice",
		GenerationDate:    "Bob",
		Fragment:          "Carol",
		Libraries: []*LibraryInfo{
			{
				URL:  "Daniel",
				Name: "Erika",
			},
		},
	}
	if clone := codeFragment.Clone(); !reflect.DeepEqual(codeFragment, clone) {
		t.Errorf("clone did not copy correctly:\n  original:  %#v\n  clone: %#v\n", codeFragment, clone)
	}
}

func TestFileClonesCorrect(t *testing.T) {
	file := &File{
		Format:      "Felix",
		ID:          "Gabrielle",
		APIName:     "Henry",
		APIVersion:  "Ilana",
		APIRevision: "Joseph",
		CodeFragment: map[string]*CodeFragment{
			"Katrina": {
				GenerationVersion: "Alice",
				GenerationDate:    "Bob",
				Fragment:          "Carol",
				Libraries: []*LibraryInfo{
					{
						URL:  "Daniel",
						Name: "Erika",
					},
				},
			},
		},
	}
	if clone := file.Clone(); !reflect.DeepEqual(file, clone) {
		t.Errorf("clone did not copy correctly:\n  original:  %#v\n  clone: %#v\n", file, clone)
	}
}

func TestPathClonesCorrect(t *testing.T) {
	path := &Path{
		APIName:         "Lawrence",
		APIVersion:      "Maria",
		SnippetRevision: "Neil",
		FragmentName:    "Olivia",
	}

	if clone := path.Clone(); !reflect.DeepEqual(path, clone) {
		t.Errorf("clone did not copy correctly:\n  original:  %#v\n  clone: %#v\n", path, clone)
	}
}

func TestInfoClonesCorrect(t *testing.T) {
	info := &Info{
		Path: Path{
			APIName:         "Lawrence",
			APIVersion:      "Maria",
			SnippetRevision: "Neil",
			FragmentName:    "Olivia",
		},
		File: File{
			Format:      "Felix",
			ID:          "Gabrielle",
			APIName:     "Henry",
			APIVersion:  "Ilana",
			APIRevision: "Joseph",
			CodeFragment: map[string]*CodeFragment{
				"Katrina": {
					GenerationVersion: "Alice",
					GenerationDate:    "Bob",
					Fragment:          "Carol",
					Libraries: []*LibraryInfo{
						{
							URL:  "Daniel",
							Name: "Erika",
						},
					},
				},
			},
		},
	}

	if clone := info.Clone(); !reflect.DeepEqual(info, clone) {
		t.Errorf("clone did not copy correctly:\n  original:  %#v\n  clone: %#v\n", info, clone)
	}
}
