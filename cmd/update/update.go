// Package update provides the top-level `Update` function to refresh and regenerate artifacts in
// discovery-artifact-manager
package update

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"time"

	"github.com/googleapis/discovery-artifact-manager/cmd/nodejs"
	"github.com/googleapis/discovery-artifact-manager/cmd/ruby"
	"github.com/googleapis/discovery-artifact-manager/pkg/common/environment"
	"github.com/googleapis/discovery-artifact-manager/pkg/common/errorlist"

	"github.com/googleapis/discovery-artifact-manager/cmd/common"
)

// update functions, run concurrently, receive a slice of absolute `fileNames` of Discovery docs from
// which to update client libraries, an absolute path to the repository `rootDirectory`, a relative
// path from there to the `subDirectory` (used by git-subrepo) for the corresponding language's client
// library, and a `remoteURL` for the client library's external repository. They return functions, to
// execute sequentially, to release the updated client libraries following a repository-wide commit
// updating all regenerated libraries. See: Update.
type update func(fileNames []string, rootDirectory, subDirectory, remoteURL string) (func() error, error)

// updater records a language client library's `Name`, its subrepository `SubDirectory` relative
// path from the repository root, the `RemoteURL` of its external repository, an `Update` function
// to regenerate it, a `Release` function to publish it (returned by the `Update` function), and any
// resulting `Error`.
type updater *struct {
	Name         string
	SubDirectory string
	RemoteURL    string
	Update       update
	Release      func() error
	Error        error
}

var updaters = []updater{
	{Name: "nodejs", Update: nodejs.Update},
	{Name: "ruby", Update: ruby.Update},
}

var rootDir string

const (
	// subDirFormat prescribes the subdirectory path pattern for client library subrepos in
	// discovery-artifact-manager, based on the library's canonical name
	subDirFormat = "clients/%s/google-api-%s-client"

	// repoURLFormat prescribes the remote URL pattern for client library repositories on GitHub,
	// based on the library's canonical name
	repoURLFormat = "https://github.com/google/google-api-%s-client"
)

func init() {
	var err error
	rootDir, err = environment.RepoRoot()
	if err != nil {
		log.Fatalf("Error locating repository root directory: %v", err)
	}

	for _, up := range updaters {
		up.SubDirectory = fmt.Sprintf(subDirFormat, up.Name, up.Name)
		up.RemoteURL = fmt.Sprintf(repoURLFormat, up.Name)
	}
}

// Update handles regeneration of all client libraries. It may `updateDiscovery` docs in the local
// cache; then invokes the sample generators and client library `Update` functions for all languages
// in `updaters`; then may `releaseLibrary` updates by performing a single repository commit
// and serially running the `Release` functions returned by the `Update` functions for all
// languages. It requires a clean initial working directory for the repository.
func Update(updateDiscovery, releaseLibrary bool) error {
	if err := common.CheckClean(rootDir); err != nil {
		return err
	}

	// Run external pulls sequentially to avoid conflicts
	for _, up := range updaters {
		if err := common.PullSubrepo(rootDir, up.SubDirectory); err != nil {
			return fmt.Errorf("Error updating %v client library: %v", up.Name, err)
		}
	}

	var now string
	var discos []string
	if updateDiscovery {
		now = time.Now().Format(time.RFC3339)
		var err error
		if discos, err = common.UpdateDiscos(); err != nil {
			return fmt.Errorf("Error updating APIs:\n%v", err)
		}
	}

	updateLibs(discos, updaters)

	// TODO(tcoffee): invoke sample generation and test

	if releaseLibrary {
		err := common.CommandIn(rootDir, "git", "commit", "-m", fmt.Sprintf(`"Regenerate from Discovery at %s"`, now)).Run()
		if err != nil {
			return fmt.Errorf("Error committing to global repository: %v", err)
		}
		err = common.CommandIn(rootDir, "git", "push", "origin").Run()
		if err != nil {
			return fmt.Errorf("Error pushing to global repository: %v", err)
		}

		releaseLibs(updaters)

		// TODO(tcoffee): invoke sample push
	}

	var errs errorlist.Errors
	for _, up := range updaters {
		if up.Error != nil {
			errs.Add(up.Error)
		}
	}
	return errs.Error()
}

// updateLibs invokes the client library `Update` functions for all languages with defined
// `updaters` for all APIs defined in Discovery `files`.
func updateLibs(files []string, updaters []updater) {
	// Ruby client library requires that created files be world-readable; setting the process umask
	// globally for library updates avoids great additional complexity.
	oldmask := syscall.Umask(0022)

	var lang sync.WaitGroup
	for _, up := range updaters {
		lang.Add(1)
		go func(up updater) {
			defer lang.Done()
			var err error
			up.Release, err = up.Update(files, rootDir, up.SubDirectory, up.RemoteURL)
			if err != nil {
				up.Error = fmt.Errorf("Error updating %v client library: %v", up.Name, err)
			}
		}(up)
	}
	lang.Wait()

	syscall.Umask(oldmask)
}

// releaseLibs invokes the client library `Release` functions for all languages with defined
// `updaters`.
func releaseLibs(updaters []updater) {
	for _, up := range updaters {
		if up.Release != nil {
			fmt.Printf("Releasing %v client library ...\n", up.Name)
			if err := up.Release(); err != nil {
				up.Error = fmt.Errorf("Error releasing %v client library: %v", up.Name, err)
			}
		}
	}
}
