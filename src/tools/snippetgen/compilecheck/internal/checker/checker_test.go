package checker

import (
	"testing"

	"gapi-cmds/src/snippetgen/common/metadata"
)

func TestLanguageExist(t *testing.T) {
	for l := range Checkers {
		if _, ok := metadata.GetLanguageFromExt(l); !ok {
			t.Errorf("extension %s has a checker but not defined in metadata", l)
		}
	}
}
