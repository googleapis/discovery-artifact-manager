package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// CommandIn returns the exec.Cmd struct to execute the named program with the given arguments in
// the specified working directory.
func CommandIn(dir, name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	return cmd
}

// CheckClean verifies that the given repository working directory contains no uncommitted changes.
func CheckClean(rootDir string) error {
	diff, err := CommandIn(rootDir, "git", "diff-index", "--quiet", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("Error verifying local repository is clean: %v", err)
	}
	if len(diff) != 0 {
		return errors.New("Local repository contains uncommitted changes")
	}
	return nil
}

// PullSubrepo pulls external changes for the given subrepository subdirectory in the given
// repository root directory, using the git-subrepo tool. It should not be run concurrently with
// other operations modifying files in the repository.
func PullSubrepo(rootDir, subDir string) error {
	if err := CommandIn(rootDir, "git", "subrepo", "pull", subDir).Run(); err != nil {
		return fmt.Errorf("Error pulling upstream library: %v", err)
	}
	return nil
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

// UpdateFile rewrites the named file in the given directory by applying the given update function
// to its contents, returning the modified contents along with any auxiliary information returned by
// the update function.
func UpdateFile(dir, filename string, update func([]byte) ([]byte, string, error)) (info string, err error) {
	pathname := path.Join(dir, filename)
	stat, err := os.Stat(pathname)
	if err != nil {
		err = fmt.Errorf("Error finding file %s: %v", pathname, err)
		return
	}
	contents, err := ioutil.ReadFile(pathname)
	if err != nil {
		err = fmt.Errorf("Error reading file %s: %v", pathname, err)
		return
	}
	changed, info, err := update(contents)
	if err != nil {
		err = fmt.Errorf("Error updating file %s: %v", pathname, err)
		return
	}
	err = ioutil.WriteFile(pathname, changed, stat.Mode())
	if err != nil {
		err = fmt.Errorf("Error writing file %s: %v", pathname, err)
		return
	}
	return
}

// ReplacePattern replaces the first instance, in the input sequence, of a pattern corresponding to a
// format string, by the given change string. It returns a non-nil error if no match appears,
// otherwise returns the modified input and the modified portion inserted by expanding any template
// variables in the change string (see: https://golang.org/pkg/regexp/).
//
// The format string is assumed to contain substitutions denoted by `%s`. The corresponding regexp
// pattern is derived by quoting regexp metacharacters and matching substitutions to shortest substrings without newlines
func ReplacePattern(input []byte, format, change string) (out []byte, inserted string, err error) {
	var pattern = regexp.MustCompile(strings.Replace(regexp.QuoteMeta(format), "%s", "(.*?)", -1) + `([\s\S]*)`)
	match := pattern.FindSubmatchIndex(input)
	if match == nil {
		err = fmt.Errorf("No match found for pattern `%s`", format)
		return
	}
	insert := pattern.Expand(nil, []byte(format), input, match)
	left, right := match[0], match[1]
	out = append(input[:left], append(insert, input[right:]...)...)
	inserted = string(insert)
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
