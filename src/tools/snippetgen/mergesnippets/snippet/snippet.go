// Package snippet implements the Merger class to retrieve and merge
// secondary and primary snippets, and to publish the merged snippets.
package snippet

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"gapi-cmds/src/common/errorlist"
	"gapi-cmds/src/common/gcs"
	"gapi-cmds/src/snippetgen/common/fragment"
	"gapi-cmds/src/snippetgen/common/metadata"
)

// fragmentMap indexes fragments by their unique key, which includes the method ID and the API version.
type fragmentMap map[fragment.Key]*fragment.Info

// Merger performs a merge between primary and secondary fragments to
// come up with the merged fragments that will be shown to users. It
// can work on purely local directories, or fetch and publish to
// Google Cloud Storage.
//
// Errors are not directly returned from the public methods, but are
// instead accumulated internally. They may be checked at any time by
// calling the Error() method on Merger. This allows multiple errors
// to be reported at once.
type Merger struct {
	// The specified locations for the primary, secondary, and
	// merged (merged) fragments.
	primaryLocation   string
	secondaryLocation string
	mergedLocation    string

	// The local working directories for the primary, secondary,
	// and merged fragments. These directories are not cleaned up
	// in any way, to allow for inspection after this tool runs.
	primaryDirectory   string
	secondaryDirectory string
	mergedDirectory    string

	// gcs is the interface to Google Cloud Storage.
	gcs *gcs.GCS

	// tmpDir is the temporary directory, should we need one.
	tmpDir string

	// The various fragments, indexed by their ID and API version
	primaryFragments   fragmentMap
	secondaryFragments fragmentMap
	mergedFragments    fragmentMap

	// errorlist is a list of accumulated errors, so that we can
	// show several to the user at once.
	errorList errorlist.Errors

	// RequestedAPIVersions is a list of the requested versioned
	// API subpaths to process. If empty, all API subpaths found
	// will be merged.
	RequestedAPIVersions []string

	// simpleMetadata indicates whether we shold should provide
	// merged snipepts with simple revision numbers rather than
	// the more complex ones that indicate their provenance.
	simpleMetadata bool
}

// Init initializes the Merger object with the provided 'gsutilPath'
// to the "gsutil" utility, and the locations of the primary,
// secondary, and merged fragments, and the temporary directory to
// use. Any errors are accumulated and can be checked with Error().
func (mrg *Merger) Init(gsutilPath, primaryLocation, secondaryLocation, mergedLocation, tmpDir string, simpleMetadata bool, apiVersions []string) {
	var err error
	if mrg.gcs, err = gcs.New(gsutilPath); err != nil {
		mrg.errorList.Add(err)
	}

	mrg.simpleMetadata = simpleMetadata
	mrg.tmpDir = tmpDir
	mrg.RequestedAPIVersions = apiVersions

	if err = mrg.createDirectories(primaryLocation, secondaryLocation, mergedLocation); err != nil {
		mrg.errorList.Add(err)
	}
}

// createDirectories creates the local primary, secondary, and merged directories that will be used for merging fragments.
func (mrg *Merger) createDirectories(primary, secondary, merged string) error {
	primaryLocation, primaryDirectory, err := mrg.prepareDirectory("primary", primary, false)
	if err != nil {
		return err
	}
	secondaryLocation, secondaryDirectory, err := mrg.prepareDirectory("secondary", secondary, false)
	if err != nil {
		return err
	}
	mergedLocation, mergedDirectory, err := mrg.prepareDirectory("merged", merged, true)
	if err != nil {
		return err
	}

	mrg.primaryLocation = primaryLocation
	mrg.secondaryLocation = secondaryLocation
	mrg.mergedLocation = mergedLocation

	mrg.primaryDirectory = primaryDirectory
	mrg.secondaryDirectory = secondaryDirectory
	mrg.mergedDirectory = mergedDirectory

	return nil
}

// prepareDirectory creates a local temporary directory 'name' if the
// 'location' location refers to GCS (rather than already being a
// local directory) or if 'createIfLocal' is set. It returns the name
// of the local directory (either the new one created or the one
// specified by 'location').
func (mrg *Merger) prepareDirectory(name, location string, createIfLocal bool) (string, string, error) {
	directory := location
	isGCS := gcs.IsGCS(location)
	if isGCS {
		if len(mrg.tmpDir) == 0 {
			tmpDir, err := ioutil.TempDir("", "snippetgen")
			if err != nil {
				return "", "", fmt.Errorf("error creating tempdir name: %s", err)
			}
			mrg.tmpDir = tmpDir
			log.Printf("created tmpDir=%q", tmpDir)
		}
		directory = filepath.Join(mrg.tmpDir, name)
	}

	if isGCS || createIfLocal {
		if err := os.MkdirAll(directory, os.ModePerm); err != nil {
			return "", "", fmt.Errorf("error creating tempdir: %s", err)
		}
	} else {
		if _, err := os.Stat(directory); err != nil {
			return "", "", fmt.Errorf("could not stat directory %q: %q: %s", name, directory, err)
		}
	}

	return location, directory, nil
}

// GetFragments obtains and reads the fragments from the primary and
// secondary locations. Failures are accumulated and can be checked via Error().
func (mrg *Merger) GetFragments() {
	mrg.pullSources()
	mrg.readFragments()
}

// pullSources retrieves the primary and secondary fragments from GCS
// to local directories if needed. It does nothing if the primary and
// secondary locations already refer to lcoal directories.
func (mrg *Merger) pullSources() {
	if gcs.IsGCS(mrg.primaryLocation) {
		mrg.transferWithGCS(mrg.primaryLocation, mrg.primaryDirectory, false)
	}

	if gcs.IsGCS(mrg.secondaryLocation) {
		mrg.transferWithGCS(mrg.secondaryLocation, mrg.secondaryDirectory, false)
	}
}

// readFragments reads the primary and secondary fragments from the
// appropriate local directories.
func (mrg *Merger) readFragments() {
	var err error
	mrg.primaryFragments, err = mrg.readFragmentsFrom(mrg.primaryDirectory)
	if err != nil {
		mrg.errorList.Add(err)
	}
	log.Printf("read %d primary fragments from %q", len(mrg.primaryFragments), mrg.primaryDirectory)

	mrg.secondaryFragments, err = mrg.readFragmentsFrom(mrg.secondaryDirectory)
	if err != nil {
		mrg.errorList.Add(err)
	}

	log.Printf("read %d secondary fragments from %q", len(mrg.secondaryFragments), mrg.secondaryDirectory)
}

// readFragmentsFrom reads all the fragment files under 'directory'
// into a fragmentMap that it returns.
func (mrg *Merger) readFragmentsFrom(directory string) (fragmentMap, error) {
	readFragments := make(fragmentMap)
	errorList := errorlist.Errors{}

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// swallow any errors due to the file not
			// being found
			return nil
		}

		if info.IsDir() {
			return nil
		}

		fragmentInfo, err := fragment.FromFile(path)
		if err != nil {
			// record error but continue processing files
			errorList.Add(err)
			return nil
		}
		if lang, fragLang := fragmentInfo.Path.Lang, metadata.FragmentLanguage; lang != fragLang {
			errorList.Add(fmt.Errorf("expected %q, found %q while reading %q", fragLang.Name, lang.Name, path))
			return nil
		}

		if err := fragmentInfo.HasConsistentMetadata(); err != nil {
			// record error but continue processing files
			errorList.Add(fmt.Errorf("inconsistent metadata in %q: %s", path, err))
			return nil
		}

		// If we have multiple revisions, use the latest one.
		key := fragmentInfo.Key()
		if previous, ok := readFragments[key]; !ok || fragmentInfo.APIRevision() > previous.APIRevision() {
			readFragments[key] = fragmentInfo
		}
		return nil
	}

	apiVersions := mrg.RequestedAPIVersions
	if len(apiVersions) == 0 {
		apiVersions = []string{""}
	}
	for _, api := range apiVersions {
		d := path.Join(directory, api)
		if err := filepath.Walk(d, walkFn); err != nil {
			return nil, fmt.Errorf("error reading snippets under %q: \n%s", d, errorList.Error())
		}
	}

	return readFragments, errorList.Error()
}

// MergeFragments merges the primary and secondary fragments, writes them to the appropriate local directory, and performs a basic validity check. Failures in any of these steps are accumulated and can be checked via Error().
func (mrg *Merger) MergeFragments() {
	mrg.computeMergedFragments()
	mrg.writeMergedFragments()
	mrg.validateMergedFragments()
}

// computeMergedFragments merges the fragments in mrg.primaryFragments
// and mrg.secondaryFragments. The former always takes precedence.
func (mrg *Merger) computeMergedFragments() {
	mrg.mergedFragments = make(fragmentMap)

	for key, file := range mrg.secondaryFragments {
		log.Printf("adding %q\n", key)
		mrg.mergedFragments[key] = file
	}

	// TODO(vchudnov): Determine whether merging a primary that
	// has no corresponding secondary should be an error.
	for key, file := range mrg.primaryFragments {
		log.Printf("merging %q\n", key)
		mergedFile, err := file.MergeWith(mrg.secondaryFragments[key], mrg.simpleMetadata)
		if err != nil {
			mrg.errorList.Add(fmt.Errorf("error merging %q: %s", key, err))
			continue
		}
		mrg.mergedFragments[key] = mergedFile
	}

	for _, fragment := range mrg.mergedFragments {
		fragment.Path.SnippetRevision = metadata.TimestampShort
	}
}

// writeMergedFragments writes the merged fragments into the
// appropriate working directories: once with the provided revision
// number in the path, and once with the "current" revision number
// sentinel in the path.
func (mrg *Merger) writeMergedFragments() {
	for key, info := range mrg.mergedFragments {
		if err := info.ToFile(mrg.mergedDirectory, false); err != nil {
			mrg.errorList.Add(fmt.Errorf("error writing file for %q: %s", key, err))
		}
		if info.Path.SnippetRevision != metadata.CurrentRevision {
			if err := info.ToFile(mrg.mergedDirectory, true); err != nil {
				mrg.errorList.Add(fmt.Errorf("error writing file at current revision for %q: %s", key, err))
			}
		}
	}
}

// validateMergedFragments performs basic integrity checks on the
// fragments, accumulating any failures so that they may be retrieved
// via Error().
func (mrg *Merger) validateMergedFragments() {
	for _, info := range mrg.mergedFragments {
		if err := info.CheckLanguages(); err != nil {
			mrg.errorList.Add(err)
		}
	}
}

// PublishMergedFragments uploads the merged fragments to GCS if appropriate.
func (mrg *Merger) PublishMergedFragments() {
	fmt.Printf("PublishMergedFragments: src=%s, dst=%s\n", mrg.mergedDirectory, mrg.mergedLocation)
	if gcs.IsGCS(mrg.mergedLocation) {
		mrg.transferWithGCS(mrg.mergedDirectory, mrg.mergedLocation, true)
	}
}

// transferWithGCS transfers 'src' to 'dst' using mrg.gcs. Either of
// the two paths may be a GCS location. The output is logged, and any
// errors are accumulated and are retrievable via Error(). If 'doAll'
// is set, only the latest revision of mrg.RequestedAPIVersions found
// under 'dst' is transferred; otherwise, all revisions under 'dst'
// are transferred.
func (mrg *Merger) transferWithGCS(src, dst string, doAll bool) {
	apiPaths, err := mrg.gcs.ListTree(src, mrg.RequestedAPIVersions)
	if err != nil {
		mrg.errorList.Add(err)
		return
	}

	if !doAll {
		apiPaths = mrg.getLatestRevisions(apiPaths)
	}
	output, err := mrg.gcs.TransferTreePaths(src, dst, true, apiPaths)
	if err != nil {
		mrg.errorList.Add(err)
	}
	log.Printf("transferWithGCS(%q, %q, %v): %v\n%s", src, dst, doAll, apiPaths, output)
}

// Error returns any errors accumulated thus far by 'mrg'.
func (mrg *Merger) Error() error {
	return mrg.errorList.Error()
}

// getLatestRevision returns a list containing the latest
// versioned-and-revisioned API path for each API. 'apiRevision' is a
// list of versioned-and-revisioned API paths which may include
// multiple revisions for a given API version. Note that a revision is
// considered more recent than another if it is lexicographically
// higher.
func (mrg *Merger) getLatestRevisions(apiRevision []string) []string {
	if len(apiRevision) == 0 {
		return apiRevision
	}

	type revisionPath struct {
		apiRevision string
		path        string
	}

	revisions := make(map[string]revisionPath)
	for _, fullPath := range apiRevision {
		revisionInfo, err := fragment.ParseFilePath(fullPath)
		if err != nil {
			mrg.errorList.Add(fmt.Errorf("error parsing revision path for %q: %s", fullPath, err))
			continue
		}
		apiKey := filepath.Join(revisionInfo.APIName, revisionInfo.APIVersion)
		thisRev := revisionInfo.SnippetRevision

		if thisRev > revisions[apiKey].apiRevision {
			revisions[apiKey] = revisionPath{thisRev, fullPath}
		}
	}
	latestRevision := make([]string, 0, len(revisions))
	for _, revPath := range revisions {
		latestRevision = append(latestRevision, revPath.path)
	}
	return latestRevision
}
