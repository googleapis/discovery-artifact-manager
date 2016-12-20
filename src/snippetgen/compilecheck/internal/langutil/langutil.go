// Package langutil provides language-independent types useful for processing code snippets and
// client libraries.
package langutil

import "strings"

// MethodInitializers maps methods to their initialization.
type MethodInitializers map[MethodID]string

// MethodParamSets maps methods to their parameters.
type MethodParamSets map[MethodID][]MethodParam

// MethodID uniquely identifies a method in a library we want to compile-check against.
// It is a subset of fragment.Path and is primarily used for map keys.
type MethodID struct {
	APIName, APIVersion, FragmentName string
}

// MethodParam represents one parameter that the client library expects.
type MethodParam struct {
	Name, Type string
}

// MethodSlice orders a slice of MethodID's by (in order) APIName, APIVersion, FragmentName.
type MethodSlice []MethodID

func (s MethodSlice) Len() int      { return len(s) }
func (s MethodSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s MethodSlice) Less(i, j int) bool {
	if c := strings.Compare(s[i].APIName, s[j].APIName); c != 0 {
		return c < 0
	}
	if c := strings.Compare(s[i].APIVersion, s[j].APIVersion); c != 0 {
		return c < 0
	}
	return s[i].FragmentName < s[j].FragmentName
}
