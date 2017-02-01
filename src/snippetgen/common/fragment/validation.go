package fragment

import (
	"fmt"

	"discovery-artifact-manager/common/errorlist"
	"discovery-artifact-manager/snippetgen/common/metadata"
)

// HasConsistentMetadata returns nil iff the metadata contained in
// info.File matches that in info.Path, and a descriptive error
// otherwise.
func (info *Info) HasConsistentMetadata() error {
	if info.Path.APIName == info.File.APIName && info.Path.APIVersion == info.File.APIVersion && info.Path.FragmentName == info.File.ID {
		return nil
	}

	return fmt.Errorf("inconsistent metadata for %q:\n  Path: %#v\n  File: %#v", info.Key(), info.Path, info.File)
}

// CheckLanguages returns an error if the languages for the code
// fragments in 'info' are incorrect. This can happen if a required
// language is missing, or if an unrecognized language is present. It
// also normalizes the languages in 'info': if any are indexed by
// their 'DisplayName', they are re-keyed by their 'Name'.
func (info *Info) CheckLanguages() error {
	allErrors := errorlist.Errors{}

	// Create a reverse map of Language.DisplayName to
	// Language.Name.
	displayLanguages := map[string]string{}
	for _, lang := range metadata.AllowedLanguages {
		if len(lang.DisplayName) != 0 {
			displayLanguages[lang.DisplayName] = lang.Name
		}
	}

	for language, s := range info.File.CodeFragment {
		if n, ok := displayLanguages[language]; ok {
			info.File.CodeFragment[n] = s
			delete(info.File.CodeFragment, language)
			continue
		}
		if _, ok := metadata.GetLanguage(language); !ok {
			allErrors.Add(fmt.Errorf("invalid language in %q: %q", info.Key(), language))
		}
	}
	for _, language := range metadata.RequiredLanguages {
		if _, ok := info.File.CodeFragment[language.Name]; !ok {
			allErrors.Add(fmt.Errorf("required language missing in %q: %q", info.Key(), language))
		}
	}
	return allErrors.Error()
}
