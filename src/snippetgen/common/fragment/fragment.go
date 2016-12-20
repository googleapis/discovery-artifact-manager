// Package fragment defines the format of the fragment files that will
// eventually be displayed on documentation pages. It also defines
// associated metadata and related functions to allow merging and
// consistency checking.
package fragment

import (
	"fmt"
	"path/filepath"
	"strconv"

	"discovery-artifact-manager/snippetgen/common/metadata"
)

// Info encapsulates information about a fragment file for a single
// API method.
type Info struct {
	// Metadata contained in the path to the file.
	Path Path

	// The parsed fragment file contents.
	File File
}

// Path contains metadata gleaned from the path to a fragment file.
type Path struct {
	APIName         string
	APIVersion      string
	SnippetRevision string
	FragmentName    string
	Lang            metadata.Language
}

func (p Path) Filename() string {
	return filepath.Join(p.APIName, p.APIVersion, p.SnippetRevision, p.FragmentName+metadata.FragmentNameSep+p.Lang.Ext)
}

// Key is the type by which to index and look up fragment files.
type Key Path

// File contains the representation of the fragment file for a single
// API method, ready for reading from or writing from JSON format.
type File struct {
	Format      string `json:"format"`
	APIName     string `json:"apiName"`
	APIVersion  string `json:"apiVersion"`
	APIRevision string `json:"apiRevision"`
	ID          string `json:"id"`

	// A map from language to the fragment exemplifying use of the
	// current method in that language.
	CodeFragment map[string]*CodeFragment `json:"codeFragment"`
}

// CodeFragment contains a snippet of code in a particular language,
// and related metadata.
type CodeFragment struct {
	GenerationVersion string `json:"generationVersion"`
	GenerationDate    string `json:"generationDate"`
	Fragment          string `json:"fragment"`

	// List of the client libraries on which this fragment
	// depends.
	Libraries []*LibraryInfo `json:"libraries"`
}

// LibraryInfo contains information about a client library assumed by
// this particular fragment.
type LibraryInfo struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

// APIRevision returns an integer representation of the API revision
// number contained in 'info.Path'. If that number cannot be parsed,
// returns a -1.
func (info *Info) APIRevision() int {
	revision, err := strconv.Atoi(info.Path.SnippetRevision)
	if err != nil {
		return -1
	}
	return revision
}

// Key returns the key that can be used to index 'info' uniquely given
// the API version and method it applies to.
func (info *Info) Key() Key {
	key := Key(info.Path)
	key.SnippetRevision = "0"
	return key
}

// String returns a text representation of 'key', useful for
// informational logging.
func (key Key) String() string {
	return fmt.Sprintf("%s~%s~%s~%s", key.APIName, key.APIVersion, key.SnippetRevision, key.FragmentName)
}
