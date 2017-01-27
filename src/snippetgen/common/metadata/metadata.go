// Package metadata contains utility functions as well as run-time constants for creating the fragment files.
package metadata

import "time"

// Language contains details about a programming language:
// its name, filename extension, and whether it is one of the required snippet languages.
type Language struct {
	Name, Ext string
	Required  bool
}

const (
	// CurrentRevision is the value of the "revision" part of the
	// fragment file path indicating this is a fragment for the
	// latest revision of the API
	CurrentRevision = "0"

	// Format in included in the fragment file to indicate how it
	// should be interpreted. This is built to future-proof the
	// clients handling fragment files in case of a format change.
	Format string = "1"

	// ExtractorVersion contains the version of this extractor
	// code, so that we can correlate extractor versions to
	// generated snippets.
	ExtractorVersion string = "0.1"

	// FragmentNameSep separates name of the fragment from the language of the fragment
	// in a file name. Base names of files are formatted like
	//   my.method.frag.py
	FragmentNameSep = ".frag."
)

var (
	// timeOfRun is the timestamp of this run, so that all
	// fragment files have the same time even if this program
	// takes some time to run.
	timeOfRun = time.Now().UTC()

	// Timestamp is the timestamp of this run for use in the
	// snippet metadata.
	Timestamp = timeOfRun.Format(time.RFC3339)

	// TimestampShort is the timestamp of this run for use in CitC
	// clients and CLs.
	TimestampShort = timeOfRun.Format("20060102-150405")

	// AllowedLanguages are the languages that are allowed in
	// snippets.  The slice is not guaranteed to be in any order.
	// Because we may skip packaging snippets in some languages
	// when the client library trails the Discovery spec, we make
	// all languages be non-required.
	//
	// TODO(vchudnov): Consider enforcing the required languages
	// after merging, but bear in mind we may not have any snippets
	// for a language at all.
	AllowedLanguages = [...]Language{
		{"Java", "java", false},
		{".NET", "cs", false},
		{"PHP", "php", false},
		{"Python", "py", false},
		{"Ruby", "rb", false},
		{"Dart", "dart", false},
		{"Go", "go", false},
		{"Google Web Toolkit", "gwt", false},
		{"Objective-C", "m", false},
		{"Node.js", "njs", false},
		{"Web", "js", false},
		FragmentLanguage,
	}

	// FragmentLanguage is the pseudo-language used for storing code fragments in GCS.
	FragmentLanguage = Language{"Code Fragment", "json", false}

	// RequiredLanguages contains languages that must be present in the published snippets.
	// The slice is not guaranteed to be in any order.
	RequiredLanguages []Language

	// langMap is a map from language name to Language struct.
	langMap = make(map[string]Language)

	// extMap is a map from language source file extension to Language struct.
	extMap = make(map[string]Language)
)

func init() {
	list := make([]Language, 0, len(AllowedLanguages))
	for _, l := range AllowedLanguages {
		if l.Required {
			list = append(list, l)
		}
		langMap[l.Name] = l
		extMap[l.Ext] = l
	}
	RequiredLanguages = list[0:len(list):len(list)]
}

// GetLanguage searches for language `name`, returning its associated Language
// and true if it is found. Otherwise, the zero value of Language and false are returned.
func GetLanguage(name string) (Language, bool) {
	l, ok := langMap[name]
	return l, ok
}

// GetLanguageFromExt searches for a language using `ext` as the filename extension,
// returning the associated Language and true if it is found.
// Otherwise, the zero value of Language and false are returned.
func GetLanguageFromExt(ext string) (Language, bool) {
	l, ok := extMap[ext]
	return l, ok
}
