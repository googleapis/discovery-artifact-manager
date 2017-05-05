package nodejs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"discovery-artifact-manager/common/errorlist"
	"discovery-artifact-manager/main/common"
)

const (
	logFile   = "CHANGELOG.md"
	logUpdate = `##### %s - %s

- Regenerated

`
)

// updateLog updates the change log file on the given path with a new version entry, and returns the version number.
func updateLog(path string) (string, error) {
	return common.UpdateFile(path, logFile, func(log []byte) (changed []byte, bumped string, err error) {
		bumped, err = common.Bump(string(log), common.Minor)
		if err != nil {
			return
		}
		today := time.Now().Format("02 January 2006")
		changed = append([]byte(fmt.Sprintf(logUpdate, bumped, today)), log...)
		return
	})
}

const packageFile = "package.json"

// updatePackage updates the package file on the given path with the given version number.
func updatePackage(path, version string) (err error) {
	_, err = common.UpdateFile(path, packageFile, func(config []byte) (changed []byte, _ string, err error) {
		changed, err = common.ReplaceValue(config, "version", version)
		return
	})
	return
}

const (
	indexFile     = "index.md"
	docLinkFormat = `* [v%s%s](http://google.github.io/google-api-nodejs-client/%s/index.html)
`
)

// updateIndex updates the documentation index on the given path with the given version number.
func updateIndex(path, version string) (err error) {
	_, err = common.UpdateFile(path, indexFile, func(index []byte) (changed []byte, _ string, err error) {
		changed, _, err = common.ReplacePattern(index, docLinkFormat,
			fmt.Sprintf(docLinkFormat, version, "$2", version)+
				fmt.Sprintf(docLinkFormat, "$1", "", "$3"))
		return
	})
	return
}

const (
	masterBranch = "master"

	// discovery-artifact-manager branch `gh-pages` corresponds to google-api-nodejs-client branch
	// `gh-pages` used for docs, and must exist
	docBranch = "gh-pages"
)

func Update(discos []string, rootDir, subDir, _ string) (release func() error, err error) {
	subPath := path.Join(rootDir, subDir)
	version, err := updateLog(subPath)
	if err != nil {
		return
	}
	err = updatePackage(subPath, version)
	if err != nil {
		return
	}

	var errs errorlist.Errors
	errChan := make(chan error, len(discos))
	done := make(chan bool)
	go func() {
		for err := range errChan {
			errs.Add(err)
		}
		done <- true
	}()
	var regen sync.WaitGroup
	for _, disco := range discos {
		regen.Add(1)
		go func(disco string) {
			defer regen.Done()
			if err := common.CommandIn(subPath, "node", "scripts/generate", disco).Run(); err != nil {
				errChan <- fmt.Errorf("Error generating library from %s: %v", disco, err)
			}
		}(disco)
	}
	regen.Wait()
	close(errChan)
	<-done
	err = errs.Error()
	if err != nil {
		err = fmt.Errorf("Error regenerating library:\n%v", err)
		return
	}

	err = common.CommandIn(subPath, "npm", "test").Run()
	if err != nil {
		err = fmt.Errorf("Error testing regenerated library: %v", err)
	}

	release = func() error {
		if err := common.CheckClean(rootDir); err != nil {
			return err
		}
		err = common.CommandIn(rootDir, "git", "subrepo", "push", subDir).Run()
		if err != nil {
			return fmt.Errorf("Error pushing to client library repository (may result from conflicting changes to remote): %v", err)
		}

		docDir := path.Join(subPath, "doc")
		err := os.RemoveAll(docDir)
		if err != nil {
			return fmt.Errorf("Error removing old docs directory %v: %v", docDir, err)
		}
		err = common.CommandIn(subPath, "npm", "run", "doc").Run()
		if err != nil {
			return fmt.Errorf("Error regenerating library docs: %v", err)
		}

		err = common.CommandIn(rootDir, "git", "checkout", docBranch).Run()
		if err != nil {
			return fmt.Errorf("Error switching to doc branch `%s`: %v", docBranch, err)
		}
		err = common.CommandIn(rootDir, "git", "subrepo", "pull", subDir, "-b", docBranch).Run()
		if err != nil {
			return fmt.Errorf("Error pulling remote doc branch `%s` (may result from conflicting changes to remote): %v", docBranch, err)
		}

		latestDir := path.Join(subPath, "latest")
		err = os.RemoveAll(latestDir)
		if err != nil {
			return fmt.Errorf("Error removing old latest docs directory %v: %v", latestDir, err)
		}
		// Go standard library does not implement simple file copying
		recentDir := path.Join(docDir, "googleapis", version)
		err = exec.Command("cp", "-r", recentDir, latestDir).Run()
		if err != nil {
			return fmt.Errorf("Error copying regenerated docs directory %v to latest directory %v: %v", recentDir, latestDir, err)
		}
		versionDir := path.Join(subPath, version)
		err = exec.Command("cp", "-r", recentDir, versionDir).Run()
		if err != nil {
			return fmt.Errorf("Error copying regenerated docs directory %v to version directory %v: %v", recentDir, versionDir, err)
		}

		err = updateIndex(subPath, version)
		if err != nil {
			return err
		}

		err = common.CommandIn(rootDir, "git", "add", "-A").Run()
		if err != nil {
			return fmt.Errorf("Error staging regenerated docs: %v", err)
		}
		err = common.CommandIn(rootDir, "git", "commit", "-m", fmt.Sprintf(`"%s"`, version)).Run()
		if err != nil {
			return fmt.Errorf("Error committing regenerated docs: %v", err)
		}
		err = common.CommandIn(rootDir, "git", "subrepo", "push", subDir, docBranch, "-b", docBranch).Run()
		if err != nil {
			return fmt.Errorf("Error pushing regenerated docs to client library repository (may result from unexpected changes to remote): %v", err)
		}
		err = common.CommandIn(rootDir, "git", "push", "origin", docBranch).Run()
		if err != nil {
			return fmt.Errorf("Error pushing regenerated docs to global repository: %v", err)
		}

		err = common.CommandIn(rootDir, "git", "checkout", masterBranch).Run()
		if err != nil {
			return fmt.Errorf("Error switching to master branch `%v`: %v", masterBranch, err)
		}

		return nil
	}
	return
}
