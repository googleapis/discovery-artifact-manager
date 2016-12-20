package fragment

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"discovery-artifact-manager/snippetgen/common/metadata"
)

// ParseFilePath extracts as much metadata as possible from a file
// path that excludes the file name.
func ParseFilePath(filePath string) (Path, error) {
	components := strings.Split(filePath, string(os.PathSeparator))
	return parseFilePathComponents(components, filePath)
}

// ParseFileName extracts the metadata from a file name.
func ParseFileName(filePath string) (Path, error) {
	components := strings.Split(filePath, string(os.PathSeparator))
	num := len(components)
	if num < 1 {
		return Path{}, fmt.Errorf("empty file path: %q", filePath)
	}
	path, err := parseFilePathComponents(components[:num-1], filePath)
	if err != nil {
		return Path{}, err
	}

	fileName := components[num-1]

	p := strings.Index(fileName, metadata.FragmentNameSep)
	if p < 0 {
		return Path{}, fmt.Errorf("cannot find separator %q: %s", metadata.FragmentNameSep, fileName)
	}
	fragName := fileName[:p]
	langExt := fileName[p+len(metadata.FragmentNameSep):]

	lang, ok := metadata.GetLanguageFromExt(langExt)
	if !ok {
		return Path{}, fmt.Errorf("unknown file extension: %s", langExt)
	}

	path.FragmentName = fragName
	path.Lang = lang

	return path, nil
}

// parseFilePathComponents constructs and returns a Path object with
// the metadata contained in the individual file path 'components',
// which exclude the fragment file name. The parameter 'filePath' is
// only used in any errors that need to be returned. This is a utility
// function used by both ParseFilePath and ParseFileName.
func parseFilePathComponents(components []string, filePath string) (Path, error) {
	num := len(components)
	if num < 3 {
		return Path{}, fmt.Errorf("too few components in path %q", filePath)
	}
	return Path{
		APIName:         components[num-3],
		APIVersion:      components[num-2],
		SnippetRevision: components[num-1],
	}, nil
}

// FromFile parses the file at 'filePath' into an Info struct that it
// returns. The 'filePath' itself is also used to extract the metadata
// for the Info struct.
func FromFile(filePath string) (*Info, error) {
	path, err := ParseFileName(filePath)
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var file File
	if err := json.Unmarshal(contents, &file); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON from %q: %s", filePath, err)
	}

	fragmentInfo := &Info{
		Path: path,
		File: file,
	}
	return fragmentInfo, nil
}

// ToFile writes 'info' into the appropriate file in a path of the
// form apiname/version/revision rooted at 'rootPath'. If
// 'markCurrent' is set, it overrides the path to have a current
// revision marker (rather than whatever revision is contained in
// Info).
func (info *Info) ToFile(rootPath string, markCurrent bool) error {
	const filePermissions = 0440
	const directoryPermissions = 0750
	if info == nil {
		return fmt.Errorf("no metadata available")
	}

	path := info.Path
	if markCurrent {
		path.SnippetRevision = metadata.CurrentRevision
	}
	filePath := filepath.Join(rootPath, path.Filename())
	directory := filepath.Dir(filePath)

	if err := os.MkdirAll(directory, directoryPermissions); err != nil {
		return err
	}

	fragmentJSON, err := json.Marshal(info.File)
	if err != nil {
		return fmt.Errorf("error while marshalling JSON for %q: %s", filePath, err)
	}

	return ioutil.WriteFile(filePath, fragmentJSON, filePermissions)
}
