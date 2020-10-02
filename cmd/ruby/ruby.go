package ruby

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/googleapis/discovery-artifact-manager/cmd/common"
)

const (
	logFile = "CHANGELOG.md"

	// TODO(tcoffee): Enhance automatic update to note added/removed APIs in change log message.
	logUpdate = `# %s
* Regenerate APIs

`
)

// updateLog updates the change log file on `path` with a new version entry, and returns the version number.
func updateLog(path string) (string, error) {
	return common.UpdateFile(path, logFile, func(log []byte) (modified []byte, bumped string, err error) {
		bumped, err = common.Bump(string(log), 3)
		if err != nil {
			return
		}
		modified = append([]byte(fmt.Sprintf(logUpdate, bumped)), log...)
		return
	})
}

const (
	versionFile   = "lib/google/apis/version.rb"
	versionFormat = `# Client library version
%sVERSION = '%s'`
)

// updateVersion updates the version file on `path` with a new `version` number.
func updateVersion(path, version string) (err error) {
	_, err = common.UpdateFile(path, versionFile, func(file []byte) (modified []byte, _ string, err error) {
		modified, _, err = common.ReplacePattern(file, versionFormat, fmt.Sprintf(versionFormat, "$1", version))
		return
	})
	return
}

const (
	masterBranch = "master"

	// discovery-artifact-manager branch `ruby_release` used for release commits for
	// google-api-ruby-client
	releaseBranch = "ruby_release"
)

// Update provides the client library update function for Ruby: see `update.update`.
func Update(fileNames []string, rootDir, subDir, repoURL string) (release func() error, err error) {
	subPath := path.Join(rootDir, subDir)
	version, err := updateLog(subPath)
	if err != nil {
		return
	}
	err = updateVersion(subPath, version)
	if err != nil {
		return
	}

	// TODO(tcoffee): make API generation concurrent by merging multiple names file outputs
	var (
		// namesOut receives Ruby names file output from client library generator
		namesOut = path.Join(subPath, "api_names_out.yaml")
		// names stores current Ruby names file from client library generator
		names = path.Join(subPath, "api_names.yaml")
		// namesGen stores current Ruby names file used by sample generator
		namesGen = path.Join(rootDir, "toolkit/src/main/resources/com/google/api/codegen/ruby/apiary_names.yaml")
	)
	// Respond to prompt: overwrite all conflicting files
	overwrite := exec.Command("echo", "a")
	generate := common.CommandIn(subPath, "bundle", "exec", "bin/generate-api",
		"gen", "generated",
		"--names_out", path.Join(subPath, "api_names_out.yaml"),
		"--file", strings.Join(fileNames, " "))
	generate.Stdin, err = overwrite.StdoutPipe()
	if err != nil {
		err = fmt.Errorf("Error establishing pipe to library generator: %v", err)
		return
	}
	err = generate.Start()
	if err != nil {
		err = fmt.Errorf("Error starting library generator: %v", err)
		return
	}
	err = overwrite.Run()
	if err != nil {
		err = fmt.Errorf("Error confirming file overwrite for library generator: %v", err)
		return
	}
	err = generate.Wait()
	if err != nil {
		err = fmt.Errorf("Error generating library: %v", err)
		return
	}

	// Go standard library does not implement simple file copying
	err = exec.Command("cp", namesOut, names).Run()
	if err != nil {
		err = fmt.Errorf("Error copying Ruby names file %v to %v: %v", namesOut, names, err)
		return
	}
	// Assume that `apiary_names.yaml` file is not modified by other client library update operations
	err = exec.Command("cp", names, namesGen).Run()
	if err != nil {
		err = fmt.Errorf("Error copying Ruby names file %v to %v: %v", names, namesGen, err)
		return
	}

	err = common.CommandIn(subPath, "bundle", "exec", "rake", "spec").Run()
	if err != nil {
		err = fmt.Errorf("Error testing regenerated library: %v", err)
		return
	}

	release = func() error {
		if err := common.CheckClean(rootDir); err != nil {
			return err
		}

		err = common.CommandIn(rootDir, "git", "checkout", "-B", releaseBranch).Run()
		if err != nil {
			return fmt.Errorf("Error switching to branch `%v` for release commit: %v", releaseBranch, err)
		}

		err = common.CommandIn(rootDir, "git", "commit", "--allow-empty", "-m", fmt.Sprintf(`"Release %s"`, version)).Run()
		if err != nil {
			return fmt.Errorf("Error creating release commit: %v", err)
		}
		// TODO(tcoffee): verify that other client libraries will not create tag name conflicts with
		// Ruby in discovery-artifact-manager repository
		err = common.CommandIn(rootDir, "git", "tag", fmt.Sprintf(`"%s"`, version)).Run()
		if err != nil {
			return fmt.Errorf("Error creating release tag: %v", err)
		}

		err = common.CommandIn(rootDir, "git", "subrepo", "push", subDir, "-b", masterBranch).Run()
		if err != nil {
			return fmt.Errorf("Error pushing to client library repository (may result from conflicting changes to remote): %v", err)
		}
		// git-subrepo does not directly support tag pushes: https://github.com/ingydotnet/git-subrepo/issues/188
		err = common.CommandIn(rootDir, "git", "push", repoURL, fmt.Sprintf("refs/subrepo/%s/push:refs/tags/%s", subDir, version)).Run()
		if err != nil {
			return fmt.Errorf("Error pushing release tag to client library repository: %v", err)
		}
		err = common.CommandIn(subDir, "gem", "push", fmt.Sprintf("pkg/*-%s.gem", version)).Run()
		if err != nil {
			return fmt.Errorf("Error pushing Ruby gem: %v", err)
		}

		err = common.CommandIn(rootDir, "git", "checkout", masterBranch).Run()
		if err != nil {
			return fmt.Errorf("Error switching to master branch `%v`: %v", masterBranch, err)
		}

		return nil
	}
	return
}
