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

// CommandIn returns the `exec.Cmd` struct to execute `program` with `arguments` in `workingDirectory`.
func CommandIn(workingDirectory, program string, arguments ...string) *exec.Cmd {
	cmd := exec.Command(program, arguments...)
	cmd.Dir = workingDirectory
	return cmd
}

// CheckClean verifies that the given repository `rootDirectory` contains no uncommitted changes.
func CheckClean(rootDirectory string) error {
	diff, err := CommandIn(rootDirectory, "git", "diff-index", "--quiet", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("Error verifying local repository is clean: %v", err)
	}
	if len(diff) != 0 {
		return errors.New("Local repository contains uncommitted changes")
	}
	return nil
}

// PullSubrepo pulls external changes for the subrepository `subDirectory` in the repository
// `rootDirectory`, using the git-subrepo tool. It should not be run concurrently with other
// operations modifying files in the repository.
func PullSubrepo(rootDirectory, subDirectory string) error {
	if err := CommandIn(rootDirectory, "git", "subrepo", "pull", subDirectory).Run(); err != nil {
		return fmt.Errorf("Error pulling upstream library: %v", err)
	}
	return nil
}

// MaxInt gives the maximum value of the machine-dependent default integer type. (Standard library
// constants are specific to machine-independent types.)
const MaxInt = int(^uint(0) >> 1)

// versionNumber groups each of the three numbers of a three-part version number '#.#.#'.
var versionNumber = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// Version finds the first three-part version number '#.#.#' in `input`, returning a slice of
// `numbers` consisting of the complete version number followed by each individual component.
func Version(input string) (numbers []string, err error) {
	numbers = versionNumber.FindStringSubmatch(input)
	if numbers == nil {
		err = errors.New("No version number '#.#.#' found")
	}
	return
}

// Bump increments a `component` of the first three-part version number '#.#.#' found in `input`,
// returning the `bumped` version string alone.
func Bump(input string, component int) (bumped string, err error) {
	num, err := Version(input)
	if err != nil {
		return
	}
	if component < 1 || component > 3 {
		err = fmt.Errorf("Invalid component %v selected for increment of version %v", component, num[0])
		return
	}
	i, err := strconv.Atoi(num[component])
	if err != nil {
		err = fmt.Errorf("Error parsing component %v of version %v: %v", component, num[0], err)
		return
	}
	if i == MaxInt {
		err = fmt.Errorf("Integer overflow incrementing component %v of version %v", component, num[0])
	}
	num[component] = strconv.Itoa(i + 1)
	bumped = strings.Join(num[1:], ".")
	return
}

// UpdateFile rewrites the file `name` in `directory` by applying an `update` function to its
// contents, returning any auxiliary `info` returned by the `update` function.
func UpdateFile(directory, name string, update func([]byte) ([]byte, string, error)) (info string, err error) {
	pathname := path.Join(directory, name)
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

// ReplacePattern replaces the first instance in `input` of a pattern corresponding to a `format`
// string, by a `change` string. It returns a non-nil error if no match appears; otherwise, it
// returns the `modified` input and the `changed` portion obtained by expanding any template
// variables in the `change` string (see: https://golang.org/pkg/regexp/).
//
// The `format` string is assumed to contain string substitutions denoted by `%s` and numeric
// substitutions denoted by `%v`. The corresponding regexp pattern is derived by quoting regexp
// metacharacters, matching string substitutions to shortest corresponding substrings without
// newlines, and matching integer substitutions to longest corresponding nonempty digit substrings.
func ReplacePattern(input []byte, format, change string) (modified []byte, changed string, err error) {
	var pattern = regexp.MustCompile(strings.Replace(strings.Replace(regexp.QuoteMeta(format),
		"%s", `(.*?)`, -1),
		"%v", `(\d+)`, -1) +
		// Capture remainder
		`([\s\S]*)`)
	match := pattern.FindSubmatchIndex(input)
	if match == nil {
		err = fmt.Errorf("No match found for pattern `%s`", format)
		return
	}
	insert := pattern.Expand(nil, []byte(change), input, match)
	// Find pattern boundaries ignoring remainder
	left, right := match[0], match[len(match)-2]
	modified = append(input[:left], append(insert, input[right:]...)...)
	changed = string(insert)
	return
}

// ReplaceValue replaces the value of a top-level `field` in a JSON `object` with a `changed` value,
// returning the `modified` object.
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
func ReplaceValue(object []byte, field string, changed interface{}) (modified []byte, err error) {
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
	ahead := io.TeeReader(bytes.NewReader(object), &behind)
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
			modified = bytes.TrimSuffix(behind.Bytes(), buffer)
			modified = append(modified, ": "...)
			modified = append(modified, serial...)
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
			modified = append(modified, buffer...)
			tail, err = ioutil.ReadAll(ahead)
			if err != nil {
				err = fmt.Errorf("Error reading JSON object: %v", err)
			}
			modified = append(modified, tail...)
			return
		}
	}
	err = fmt.Errorf("Top-level field %#v not found in JSON object", field)
	return
}
