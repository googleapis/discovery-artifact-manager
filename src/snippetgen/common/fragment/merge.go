package fragment

import (
	"fmt"
	"strings"

	"discovery-artifact-manager/snippetgen/common/metadata"
)

// currentMergeVersion contains the version identifier of the current
// merging algorithm. This identifier will be included in merged
// CodeFragments.GenerationVersion
const currentMergeVersion = "1"

// simpleMetadata is the snippet revision suffix used, when the
// simpleMetadata option is set, to indicate that a given primary code
// snippet did not have a secondary code snippet with which to merge.
const simpleMetadataPrimarySuffix = ".p"

// MergeWith merges 'info' with 'other', given preference to the
// former, and returns the result. This means that if a CodeFragment
// for any given language is present in 'info', it is used; otherwise,
// the CodeFragment for that language in 'other' is used. The Key()
// result on 'info', 'other' and the merge result must be identical,
// or an error will occur. For each language, if 'simpleMetadata' is
// true, the snippet metadata from the source that winds up being used
// in the merge result is copied verbatim. If 'simpleMetadata' is
// false, the snippet metadata reflects that of both snippets that
// were compared.
func (info *Info) MergeWith(other *Info, simpleMetadata bool) (*Info, error) {
	if info == nil && other == nil {
		return nil, nil
	}

	if other == nil {
		merged := info.Clone()
		for language, codeFragment := range merged.File.CodeFragment {
			merged.File.CodeFragment[language].updateMergedMetadata(codeFragment, nil, simpleMetadata)
		}
		merged.File.APIRevision = mergedAPIRevision(merged.File.APIRevision, "", simpleMetadata)
		return merged, nil
	}

	if info == nil {
		merged := other.Clone()
		for language, codeFragment := range merged.File.CodeFragment {
			merged.File.CodeFragment[language].updateMergedMetadata(nil, codeFragment, simpleMetadata)
		}
		merged.File.APIRevision = mergedAPIRevision("", merged.File.APIRevision, simpleMetadata)
		return merged, nil
	}

	thisFile := info.File
	otherFile := other.File

	if thisFile.Format != otherFile.Format {
		return nil, fmt.Errorf("different fragment formats when merging %q (%q) and %q (%q)", info.Key(), thisFile.Format, other.Key(), otherFile.Format)
	}
	if !AreCommensurate(&info.File, &other.File) {
		return nil, fmt.Errorf("trying to merge disparate fragments %q and %q\n%q:%#v\n%q:%#v", info.Key(), other.Key(), info.Key(), info, other.Key(), otherFile)
	}

	merged := other.Clone()
	for language, codeFragment := range thisFile.CodeFragment {
		// Treat an empty fragment as being non-existent. Note
		// that non-empty whitespace is significant, though,
		// and still overrides the corresponding Fragment in
		// otherFile.
		if len(codeFragment.Fragment) == 0 {
			continue
		}

		otherFragment, _ := otherFile.CodeFragment[language]
		mergedFragment := codeFragment.Clone()
		mergedFragment.updateMergedMetadata(codeFragment, otherFragment, simpleMetadata)
		merged.File.CodeFragment[language] = mergedFragment
	}

	merged.Path.SnippetRevision = moreRecentSnippetRevision(info.Path.SnippetRevision, merged.Path.SnippetRevision)
	if simpleMetadata {
		merged.File.APIRevision = thisFile.APIRevision
	} else {
		merged.File.APIRevision = mergedAPIRevision(thisFile.APIRevision, otherFile.APIRevision, simpleMetadata)
	}

	if merged.Key() != info.Key() || merged.Key() != other.Key() {
		return nil, fmt.Errorf("consistency error: the merged key does not match the inputs: info: %q,  other: %q, merged: %q", info.Key(), other.Key(), merged.Key())
	}

	return merged, nil
}

// moreRecentSnippetRevision returns the most recent revision of 'a' and
// 'b'. It assumes that lexicographically higher values are more
// recent.
func moreRecentSnippetRevision(a, b string) string {
	if a > b {
		return a
	}
	return b
}

// updateMergedMetadata updates the field of the merged 'cf' with a
// merged value generated from 'primary' and 'secondary' if
// 'simpleMetadata' is false. If it is true, the fields corresponding
// to the first of 'primary'' or 'secondary'' that is non-null are
// copied over to 'cf'.'
func (cf *CodeFragment) updateMergedMetadata(primary, secondary *CodeFragment, simpleMetadata bool) {
	if primary == nil && secondary == nil {
		return
	}
	if simpleMetadata {
		src := primary
		if src == nil {
			src = secondary
		}
		cf.GenerationVersion = src.GenerationVersion
		cf.GenerationDate = src.GenerationDate
		return
	}
	primaryGenerationVersion := ""
	primaryGenerationDate := ""
	secondaryGenerationVersion := ""
	secondaryGenerationDate := ""
	if primary != nil {
		primaryGenerationVersion = primary.GenerationVersion
		primaryGenerationDate = primary.GenerationDate
	}
	if secondary != nil {
		secondaryGenerationVersion = secondary.GenerationVersion
		secondaryGenerationDate = secondary.GenerationDate
	}

	cf.GenerationVersion = fmt.Sprintf("%s[%s(%s)]+[%s(%s)]", currentMergeVersion, primaryGenerationVersion, primaryGenerationDate, secondaryGenerationVersion, secondaryGenerationDate)
	cf.GenerationDate = metadata.Timestamp
}

// mergedAPIRevision returns a string with the API revision to use for
// merged fragments.
func mergedAPIRevision(primary, secondary string, simpleMetadata bool) string {
	if !simpleMetadata {
		return fmt.Sprintf("%s~%s", primary, secondary)
	}
	if len(primary) == 0 {
		if strings.HasSuffix(secondary, simpleMetadataPrimarySuffix) {
			return secondary
		}
		return fmt.Sprintf("%s%s", secondary, simpleMetadataPrimarySuffix)
	}
	return primary
}

// AreCommensurate returns true iff 'first' and 'second' describe the
// same method at the same API version such that the concept of
// merging them makes sense.
func AreCommensurate(first, second *File) bool {
	return (first != nil &&
		second != nil &&
		first.Format == second.Format &&
		first.ID == second.ID &&
		first.APIName == second.APIName &&
		first.APIVersion == second.APIVersion)
}
