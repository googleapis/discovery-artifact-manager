package common

import (
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
)

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
