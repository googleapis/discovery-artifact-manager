// Package main contains the main driver for the mergesnippets tool,
// which merges manual and automatic snippets into public snippets
// that can be published to GCS.
//
// The simplest invocation is:
//
//    main --primary=PATH1 --secondary=PATH2 --merged=PATH3 [APISUBPATHS...]
//
// where APISUBPATHS are specific subpaths under both PATH1 and PATH2
// that identify the content to merge.
//
// Note that since this is a general tool, it does not delete the
// contents of PATH3 before writing to it.
package main

import (
	"flag"
	"log"

	"discovery-artifact-manager/snippetgen/mergesnippets/snippet"
)

var (
	primaryLocation   = flag.String("primary", "manual/", "Location of the manual snippets to merge. If the prefix is gcs://, the location is taken to be in GCS; otherwise, the location is interpreted as a local path.")
	secondaryLocation = flag.String("secondary", "automatic/", "Location of the automatic snippets to merge. If the prefix is gcs://, the location is taken to be in GCS; otherwise, the location is interpreted as a local path.")
	mergedLocation    = flag.String("merged", "public/", "Location of the public, merged snippets. If the prefix is gcs://, the location is taken to be in GCS; otherwise, the location is interpreted as a local path.")
	gsutilPath        = flag.String("gsutil", "gsutil", "Path to the gsutil command (if not in your $PATH)")
	tmpDir            = flag.String("tmp", "", "Path under which to create temporary directories")
	simpleMetadata    = flag.Bool("simple_metadata", false, "Whether to have simple metadata in the merged artifact rather than metadata that traces both merge sources")
	currentOnly       = flag.Bool("current_only", false, "Whether to only merge API versions at the current revision")
)

func main() {
	flag.Parse()
	mrg := &snippet.Merger{}

	mrg.Init(*gsutilPath, *primaryLocation, *secondaryLocation, *mergedLocation, *tmpDir, *simpleMetadata, *currentOnly, flag.Args())
	if err := mrg.Error(); err != nil {
		log.Fatalf("initialization errors:\n%s", err)
	}

	mrg.GetFragments()

	mrg.MergeFragments()
	if err := mrg.Error(); err != nil {
		log.Fatalf("merging errors:\n%s", err)
	}

	mrg.PublishMergedFragments()
	if err := mrg.Error(); err != nil {
		log.Fatalf("publishing errors:\n%s", err)
	}
}
