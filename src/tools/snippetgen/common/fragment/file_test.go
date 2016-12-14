package fragment

import (
	"testing"

	"gapi-cmds/src/snippetgen/common/metadata"
)

func TestParseFileName(t *testing.T) {
	cases := []struct {
		fileName  string
		wantPath  Path
		wantError bool
	}{
		{
			fileName: "intelligence/v3/1415/smart.actions.think.frag.json",
			wantPath: Path{
				APIName:         "intelligence",
				APIVersion:      "v3",
				SnippetRevision: "1415",
				FragmentName:    "smart.actions.think",
				Lang: metadata.Language{
					Name: "Code Fragment",
					Ext:  "json",
				},
			},
		},
		{
			fileName:  "v3/1415/smart.actions.think.frag.json",
			wantPath:  Path{},
			wantError: true,
		},
		{
			fileName:  "intelligence/v3/1415/smart.actions.think.txt.json",
			wantPath:  Path{},
			wantError: true,
		},
		{
			fileName:  "intelligence/v3/1415/smart.actions.think.frag.cobol",
			wantPath:  Path{},
			wantError: true,
		},
	}

	for _, c := range cases {
		p, err := ParseFileName(c.fileName)
		if got, want := err != nil, c.wantError; got != want {
			t.Errorf("%q error: got: %v (%s), want %v", c.fileName, got, err, want)
		}
		if got, want := p, c.wantPath; got != want {
			t.Errorf("%q: unexpected output Path:\n   got: %#v\n  want: %#v", c.fileName, got, want)
		}
	}
}
