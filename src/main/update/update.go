package update

import (
	"fmt"
	"log"
	"sync"
	"time"

	"discovery-artifact-manager/common/environment"
	"discovery-artifact-manager/common/errorlist"
	"discovery-artifact-manager/main/common"
	"discovery-artifact-manager/main/nodejs"
)

var rootDir string

func init() {
	var err error
	rootDir, err = environment.RepoRoot()
	if err != nil {
		log.Fatal("Error locating repository root directory: %v", err)
	}
}

// update functions receive a slice of absolute file names of Discovery docs from which to update
// client libraries, an absolute path to the repository root directory, a relative path from there
// to the subdirectory (used by git-subrepo) for the corresponding language's client library, and a
// read/write mutex for operations dependent on the repository file system state (e.g., Git
// commands). They return functions to execute to release the updated client libraries
// following a repository-wide commit updating all regenerated libraries. See: Update.
type update func(discos []string, rootDir, subDir string, repo *sync.RWMutex) (func() error, error)

type updater *struct {
	Lib     string
	SubDir  string
	Update  update
	Release func() error
	Error   error
}

// updaters maps language library names to the relative path from the repository root to its
// subdirectory, a function regenerating the client library from the local Discovery doc cache, and
// a function thence returned releasing the updated client library. See: Update.
var updaters = []updater{
	{Lib: "nodejs", SubDir: "clients/nodejs/google-api-nodejs-client", Update: nodejs.Update},
}

var repo = &sync.RWMutex{}
var lang sync.WaitGroup

// Update updates the local Discovery doc cache, if indicated; then invokes the sample generators
// and the client library Update functions for all languages in updaters; then, if indicated,
// performs a single repository commit updating all client libraries and runs the client library
// release functions returned by the Update functions for all languages. It requires a clean initial
// working directory for the repository.
func Update(updateDisco, releaseLib bool) error {
	if err := common.CheckClean(rootDir); err != nil {
		return err
	}

	var now string
	var discos []string
	if updateDisco {
		now = time.Now().Format(time.RFC3339)
		var err error
		if discos, err = common.UpdateDiscos(); err != nil {
			return fmt.Errorf("Error updating APIs:\n%v", err)
		}
	}

	updateLibs(discos, updaters)

	// TODO(tcoffee): invoke sample generation and test

	if releaseLib {
		// TODO(tcoffee): git commit from rootDir
		err := common.CommandIn(rootDir, "git", "commit", "-m", `"Regenerate from Discovery at `+now+`"`).Run()
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

// updateLibs invokes the client library Update functions for all languages in updaters.
func updateLibs(discos []string, updaters []updater) {
	for _, up := range updaters {
		lang.Add(1)
		go func(up updater) {
			defer lang.Done()
			var err error
			up.Release, err = up.Update(discos, rootDir, up.SubDir, repo)
			if err != nil {
				up.Error = fmt.Errorf("Error updating %v client library: %v", up.Lib, err)
			}
		}(up)
	}
	lang.Wait()
}

// releaseLibs invokes the client library Release functions for all languages in updaters.
func releaseLibs(updaters []updater) {
	for _, up := range updaters {
		lang.Add(1)
		go func(up updater) {
			defer lang.Done()
			if up.Release != nil {
				if err := up.Release(); err != nil {
					up.Error = fmt.Errorf("Error releasing %v client library: %v", up.Lib, err)
				}
			}
		}(up)
	}
	lang.Wait()
}
