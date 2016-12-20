package metadata

import "testing"

func TestGetLanguage(t *testing.T) {
	for _, l := range AllowedLanguages {
		if g, ok := GetLanguage(l.Name); !ok {
			t.Errorf("language defined but not found: %s", l.Name)
		} else if g != l {
			t.Errorf("found wrong language, expected %v, found %v", l, g)
		}
	}
}

func TestNoLanguage(t *testing.T) {
	langs := [...]string{"foobar language"}
	for _, l := range langs {
		if _, exist := GetLanguage(l); exist {
			t.Errorf("language found but should not exist: %s", l)
		}
	}
}

func TestRequiredLanguages(t *testing.T) {
	for _, l := range RequiredLanguages {
		if !l.Required {
			t.Errorf("language is required but not marked required: %s", l.Name)
		}
		if g, ok := GetLanguage(l.Name); !ok {
			t.Errorf("language required but not defined: %s", l.Name)
		} else if l != g {
			t.Errorf("required language different from the definition: %s", l.Name)
		}
	}

	for _, l := range AllowedLanguages {
		found := false
		for _, r := range RequiredLanguages {
			if l.Name == r.Name {
				found = true
				break
			}
		}
		if l.Required && !found {
			t.Errorf("language marked required but not in RequiredLanguages: %s", l.Name)
		}
	}
}

func TestGetLanguageFromExt(t *testing.T) {
	for _, l := range AllowedLanguages {
		if g, ok := GetLanguageFromExt(l.Ext); !ok {
			t.Errorf("cannot look up extension: %s", l.Ext)
		} else if l != g {
			t.Errorf("language different from definition: %s", l.Name)
		}
	}
}

func TestNoLanguageFromExt(t *testing.T) {
	langs := [...]string{"foo", "bar"}
	for _, l := range langs {
		if _, exist := GetLanguageFromExt(l); exist {
			t.Errorf("language found but should not exist: %s", l)
		}
	}
}
