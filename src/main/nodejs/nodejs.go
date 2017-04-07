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

func updateLog(path string) (version string, err error) {
	bumped, err := common.UpdateFile(path, logFile, func(log []byte) (changed []byte, bumped interface{}, err error) {
		bumped, err = common.Bump(string(log), common.Minor)
		if err != nil {
			return
		}
		today := time.Now().Format("02 January 2006")
		changed = append([]byte(fmt.Sprintf(logUpdate, bumped, today)), log...)
		return
	})
	version = bumped.(string)
	return
}

const packageFile = "package.json"

func updatePackage(path, version string) (err error) {
	_, err = common.UpdateFile(path, packageFile, func(config []byte) (changed []byte, _ interface{}, err error) {
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

// docLinkPattern matches entries in the doc index, plus an unconstrained suffix to allow regexp
// replacement using standard library functions, which do not support single replacement.
var docLinkPattern = regexp.MustCompile(strings.Replace(regexp.QuoteMeta(docLinkFormat), "%s", "(.*?)", -1) + `([\s\S]*)`)

func updateIndex(path, version string) (err error) {
	_, err = common.UpdateFile(path, indexFile, func(index []byte) (changed []byte, _ interface{}, err error) {
		changed = docLinkPattern.ReplaceAll(index, []byte(
			fmt.Sprintf(docLinkFormat, version, "$2", version)+
				fmt.Sprintf(docLinkFormat, "$1", "", "$3")))
		if bytes.Equal(changed, index) {
			err = fmt.Errorf("No match found for doc link %s", docLinkFormat)
		}
		return
	})
	return
}

// discovery-artifact-manager branch `gh-pages` corresponds to google-api-nodejs-client branch
// `gh-pages` used for docs
const docBranch = "gh-pages"

func Update(discos []string, rootDir, subDir string, repo *sync.RWMutex) (release func() error, err error) {
	defer repo.Unlock()
	pull := common.CommandIn(rootDir, "git", "subrepo", "pull", subDir)
	repo.Lock()
	err = pull.Start()
	if err != nil {
		err = fmt.Errorf("Error starting upstream library pull: %v", err)
		return
	}
	err = pull.Wait()
	repo.Unlock()
	if err != nil {
		err = fmt.Errorf("Error pulling upstream library: %v", err)
		return
	}

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
		repo.Lock()
		defer repo.Unlock()

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
		err = common.CommandIn(rootDir, "git", "commit", "-m", `"`+version+`"`).Run()
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

		err = common.CommandIn(rootDir, "git", "checkout", "master").Run()
		if err != nil {
			return fmt.Errorf("Error switching to master branch: %v", err)
		}

		return nil
	}
	return
}
