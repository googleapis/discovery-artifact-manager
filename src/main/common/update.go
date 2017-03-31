package common

import (
	"discovery-artifact-manager/common/environment"
	"discovery-artifact-manager/common/errorlist"
	"discovery-artifact-manager/main/common"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// timeFormat modifies the standard RFC3339 format string to make legal for git branch names
const timeFormat = strings.Replace(time.RFC3339, ":", ".")

// update functions receive an absolute path to the repository root directory, a relative path from
// there to the subdirectory (used by git-subrepo) for the corresponding language's client library,
// and mutex for operations dependent on the process's current working directory (e.g., spawning
// external commands), and a read/write mutex for operations dependent on the file system state
// (e.g., Git commands). They return a function to execute to release the updated client library
// following a repository-wide commit updating all regenerated libraries. See: Update.
type update func(rootDir, subDir string, cwdops *sync.Mutex, fileops *sync.RWMutex) (func() error, error)

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

// Update updates the local Discovery doc cache, if indicated; then invokes the sample generators
// and the client library Update functions for all languages in updaters; then, if indicated,
// performs a single repository commit updating all client libraries and runs the client library
// release functions returned by the Update functions for all languages. It requires a clean initial
// working directory for the repository.
func Update(updateDisco, releaseLib bool) error {
	if err := checkClean(); err != nil {
		return err
	}

	var now string
	if updateDisco {
		now = time.Now().Format(timeFormat)
		if err := common.UpdateDiscos(); err != nil {
			return fmt.Errorf("Error updating APIs:\n%v", err)
		}
	}

	updateLibs(updaters)

	// TODO(tcoffee): invoke sample generation and test

	if releaseLib {
		err = os.Chdir(rootDir)
		if err != nil {
			return fmt.Errorf("Error switching to root directory %s: %v", rootDir, err)
		}

		// TODO(tcoffee): git commit

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

// checkClean verifies that the repository working directory contains no uncommitted changes.
func checkClean() error {
	diff, err := exec.Command("git", "diff-index", "--quiet", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("Error verifying local repository is clean: %v", err)
	}
	if diff.Len() != 0 {
		return errors.New("Local repository contains uncommitted changes")
	}
	return nil
}

// updateLibs invokes the client library Update functions for all languages in updaters.
func updateLibs() {
	rootDir, err := environment.RepoRoot()
	if err != nil {
		return fmt.Errorf("Error locating repository root directory: %v", err)
	}
	var cwdOps = &sync.Mutex{}
	var fileOps = &sync.RWMutex{}

	var lang sync.WaitGroup
	for _, up := range updaters {
		lang.Add(1)
		go func(up updater) {
			defer lang.Done()
			up.Release, err = up.Update(rootDir, up.SubDir, cwdOps, fileOps)
			if err != nil {
				up.Error = fmt.Errorf("Error updating %v client library: %v", up.Lib, err)
			}
		}(up)
	}
	lang.Wait()
}

// releaseLibs invokes the client library Release functions for all languages in updaters.
func releaseLibs() {
	var lang sync.WaitGroup
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

// MaxInt gives the maximum value of the machine-dependent default integer type. (Standard library
// constants are specific to machine-independent types.)
const MaxInt = int(^uint(0) >> 1)

// Major, Minor, and Patch define group indices of corresponding version number components in
// versionNumber. Indices are one-based to match subgroup indices in regexp capture.
const (
	_ = iota
	Major
	Minor
	Patch
)

// versionNumber groups each of the three numbers of a three-part version number '#.#.#'.
var versionNumber = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// Bump increments the specified component of the first three-part version number '#.#.#' found in
// the input, returning the incremented version string alone.
func Bump(versioned string, component int) (bumped string, err error) {
	nums := versionNumber.FindStringSubmatch(versioned)
	if nums == nil {
		err = errors.New("No existing version number '#.#.#' found")
		return
	}
	if component < Major || component > Patch {
		err = fmt.Errorf("Invalid component %v selected for increment of version %v", component, nums[0])
		return
	}
	i, err := strconv.Atoi(nums[component])
	if err != nil {
		err = fmt.Errorf("Error parsing component %v of version %v: %v", component, nums[0], err)
		return
	}
	if i == MaxInt {
		err = fmt.Errorf("Integer overflow incrementing component %v of version %v", component, nums[0])
	}
	nums[component] = strconv.Itoa(i + 1)
	bumped = strings.Join(nums[Major:], ".")
	return
}

// UpdateFile rewrites the named file by applying the given update function to its contents,
// returning the modified contents along with any auxiliary information returned by the update
// function.
func UpdateFile(name string, update func([]byte) ([]byte, interface{}, error)) (info interface{}, err error) {
	stat, err := os.Stat(name)
	if err != nil {
		err = fmt.Errorf("Error finding file %s: %v", name, err)
		return
	}
	contents, err := ioutil.ReadFile(name)
	if err != nil {
		err = fmt.Errorf("Error reading file %s: %v", name, err)
		return
	}
	changed, info, err := update(contents)
	if err != nil {
		err = fmt.Errorf("Error updating file %s: %v", name, err)
		return
	}
	err = ioutil.WriteFile(name, changed, stat.Mode())
	if err != nil {
		err = fmt.Errorf("Error writing file %s: %v", name, err)
		return
	}
	return
}

// ReplaceValue replaces the value of a top-level field in a JSON object with the given changed
// value, returning the modified object.
//
// This implementation works around the lack of order-preserving (un)marshaling in the standard
// library `json` package, by the most straightforward means available for our purposes: It uses the
// standard library decoder to locate the boundaries of the existing value, replacing it with an
// encoding of the changed value with no extraneous formatting, and leaving the remainder of the
// object encoding unchanged.
//
// Note that Go developers have rejected order-preserving capabilities in the standard library with
// assertions that JSON is not a human-readable format (the standard library's support for
// indentation notwithstanding). Compare the second sentence of http://www.json.org, which states
// that "[JSON] is easy for humans to read and write."
func ReplaceValue(in []byte, field string, changed interface{}) (out []byte, err error) {
	malformed := func(err error) error {
		if err != nil {
			return fmt.Errorf("Error parsing JSON object: %v", err)
		} else {
			return errors.New("Error parsing JSON object")
		}
	}

	serial, err := json.Marshal(changed)
	if err != nil {
		err = fmt.Errorf("Error encoding %#v to JSON: %v", changed, err)
		return
	}

	var behind bytes.Buffer
	ahead := io.TeeReader(bytes.NewReader(in), &behind)
	parse := json.NewDecoder(ahead)

	// Enter top-level object
	open, err := parse.Token()
	if err != nil || open != json.Delim('{') {
		err = malformed(err)
		return
	}

	var token json.Token
	var buffer, tail []byte
	for parse.More() {

		// Read next top-level name
		token, err = parse.Token()
		name, ok := token.(string)
		if err != nil || !ok {
			err = malformed(err)
			return
		}
		hit := (name == field)
		if hit {
			buffer, err = ioutil.ReadAll(parse.Buffered())
			if err != nil {
				err = fmt.Errorf("Error reading JSON decoder buffer: %v", err)
				return
			}
			out = bytes.TrimSuffix(behind.Bytes(), buffer)
			out = append(out, ": "...)
			out = append(out, serial...)
		}

		// Read next top-level value
		var value json.RawMessage
		err = parse.Decode(&value)
		if err != nil {
			err = malformed(err)
			return
		}
		if hit {
			buffer, err = ioutil.ReadAll(parse.Buffered())
			if err != nil {
				err = fmt.Errorf("Error encoding %#v to JSON: %v", changed, err)
				return
			}
			out = append(out, buffer...)
			tail, err = ioutil.ReadAll(ahead)
			if err != nil {
				err = fmt.Errorf("Error reading JSON object: %v", err)
			}
			out = append(out, tail...)
			return
		}
	}
	err = fmt.Errorf("Top-level field %#v not found in JSON object", field)
	return
}
