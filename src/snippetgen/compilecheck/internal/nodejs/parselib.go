package nodejs

import (
	"bufio"
	"bytes"
	"path/filepath"
	"regexp"
	"unicode"

	"discovery-artifact-manager/snippetgen/compilecheck/internal/filesys"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/langutil"
)

// parseLibs parses libraries referenced by `sampleMethods`. It expects the libraries to be under
// `libDir` and opens the files with `opener`. It returns a map of methods to their signatures and
// any errors.
func parseLibs(methodInits langutil.MethodInitializers, libDir string, opener filesys.Opener) (langutil.MethodParamSets, error) {
	type libID struct {
		apiName, apiVersion string
	}

	libs := make(map[libID]bool, len(methodInits))
	for mid := range methodInits {
		libs[libID{
			apiName:    mid.APIName,
			apiVersion: mid.APIVersion,
		}] = true
	}

	params := make(langutil.MethodParamSets)
	for l := range libs {
		if err := parseLib(libDir, l.apiName, l.apiVersion, params, opener); err != nil {
			return nil, err
		}
	}
	return params, nil
}

var (
	docCommentStart = []byte("/**")
	docCommentEnd   = []byte("*/")
	docCommentParam = regexp.MustCompile(`@param\s+\{(\w+)\}\s+params\.(\S+)`)
)

// parse states used by parseLib
const (
	parseText = iota
	parseName
	parseParams
)

// typeConvert converts types in the API documentation to JavaScript types.
var typeConvert = map[string]string{
	"integer": "number",
}

// parseLib is a helper of parseLibs, parsing only one file. It updates `params` with the content of
// the file.
func parseLib(libDir, apiName, apiVersion string, params langutil.MethodParamSets, opener filesys.Opener) error {
	file, err := opener.Open(filepath.Join(libDir, apiName, apiVersion+".js"))
	if err != nil {
		return err
	}
	defer file.Close()

	state := parseText
	sc := bufio.NewScanner(file)
	var methodName string
	var currentParams []langutil.MethodParam

	// Parse the function comment, which looks like:
	//
	// /**
	//  * myApi.myMethod
	//  * @param {object} params Description of params
	//  * @param {string} params.oneField Description of params.oneField
	//  * @param {object} params.otherField Description of params.otherField
	//  * @param {object=} params.optionalField Description of params.optionalField
	//  * @param {function} callback Description of callback
	//  */
	//
	// We only care about required fields in the `params` object, so we drop other parameters and
	// optional fields. We require that the '@param' declaration, the type, and
	// the parameter name all appear on the same line. Descriptions may span multiple lines, however.
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		switch {
		case state == parseText && bytes.Equal(line, docCommentStart):
			state = parseName
		case state == parseName:
			if bytes.IndexRune(line, '*') != 0 {
				state = parseText
				break
			}
			line = bytes.TrimSpace(line[1:])
			// Check if the comment looks like a comment for methods or not.
			// Comments for class usually have spaces and don't have dots; ignore those.
			if bytes.IndexFunc(line, unicode.IsSpace) < 0 && bytes.IndexRune(line, '.') >= 0 {
				methodName = string(line)
				state = parseParams
			} else {
				state = parseText
			}
		case state == parseParams && bytes.Equal(line, docCommentEnd):
			params[langutil.MethodID{
				APIName:      apiName,
				APIVersion:   apiVersion,
				FragmentName: methodName,
			}] = append([]langutil.MethodParam{}, currentParams...)
			currentParams = currentParams[:0]
			state = parseText
		case state == parseParams:
			if match := docCommentParam.FindSubmatch(line); len(match) > 0 {
				typ := string(match[1])
				if t, ok := typeConvert[typ]; ok {
					typ = t
				}
				currentParams = append(currentParams, langutil.MethodParam{
					Name: string(match[2]),
					Type: typ,
				})
			}
		}
	}
	return sc.Err()
}
