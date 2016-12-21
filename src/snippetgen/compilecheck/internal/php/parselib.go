package php

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"discovery-artifact-manager/snippetgen/compilecheck/internal/filesys"
)

// ParameterPath contains the path (class->method->parameter) used to identify the corresponding
// parameter type.
type ParameterPath struct {
	ClassName, MethodName, ParameterName string
}

// PathTypePair represents the parsed parameter information (name and types) of methods in the
// classes. The structure looks like this:
// Path  unique path to the parameter (class x method x parameter name)
// Types parameter types array (Some parameters may allow more than one type)
type PathTypePair struct {
	Path  ParameterPath
	Types []string
}

// ParsedLib is an array of PathTypePair which represents the parameter information parsed from the
// given library.
type ParsedLib []PathTypePair

// parseLibs parses the libraries used by the PHP snippets. It expects the libraries to be under
// `libDir` and opens the files with `opener`.
// It returns a ParsedLib object that contains parameters and their types for each method of
// all services under `libDir`. For more information please see comments of `ParsedLib`.
func parseLibs(libDir string, opener filesys.Opener) (ParsedLib, error) {
	var parsedLib ParsedLib
	err := filepath.Walk(libDir, filesParser(&parsedLib, opener))
	return parsedLib, err
}

// filesParser returns a walk function that is passed to `filepath.Walk` and which iterates through
// the given directory and parses containing files using `parseFile` method.
func filesParser(parsedLib *ParsedLib, opener filesys.Opener) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("unable to parse directory")
		}
		if !info.IsDir() {
			return parseFile(parsedLib, path, opener)
		}
		return nil
	}
}

// Patterns to identify the types of lines in the PHP files.
const (
	classSignatureLine  = "class "
	methodSignatureLine = "public function "
	paramLine           = "@param"
)

// parseFile parses the file that contains service classes and comments and stores the
// data into the `parsedLib` object.
// In the context of compile check, only the class name, method name, and parameter comments
// are parsed. The format should look like:
//
// class ClassName extends Google_Service_Resource
// {
//   /**
//    * Description
//    *
//    * @param string $paramA description
//    * ...
//    */
//   public function methodName($paramA, $paramB, $optParams = array())
//   {
//     ...
//   }
//   ...
// }
//
func parseFile(parsedLib *ParsedLib, path string, fileOpener filesys.Opener) error {
	file, err := fileOpener.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// A map that maps param name to its possible types.
	var params = make(map[string][]string)
	var className string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, classSignatureLine) {
			className = parseClassName(line)
		} else if strings.Contains(line, paramLine) {
			paramName, paramTypes := parseParam(line)
			params[paramName] = paramTypes
		} else if strings.HasPrefix(line, methodSignatureLine) {
			methodName := parseMethodName(line)
			for paramName, paramTypes := range params {
				paramPath := ParameterPath{
					ClassName:     className,
					MethodName:    methodName,
					ParameterName: paramName,
				}
				*parsedLib = append(*parsedLib, PathTypePair{
					Path:  paramPath,
					Types: paramTypes,
				})
			}

			// Reset params
			params = make(map[string][]string)
		}
	}
	return scanner.Err()
}

// parseClassName parses the class name from the given PHP line.
func parseClassName(line string) string {
	lineParts := splitPHPLine(line)
	if len(lineParts) > 1 {
		return lineParts[1]
	}
	return ""
}

// parseMethodName parses the method name from the given PHP line.
func parseMethodName(line string) string {
	lineParts := splitPHPLine(line)
	if len(lineParts) > 2 {
		return lineParts[2]
	}
	return ""
}

// renamedTypes is a map of Discovery data type names to the corresponding PHP data types.
var renamedTypes = map[string]string{
	"int":  "integer",
	"bool": "boolean",
}

// parseParam parses the parameter name and type from the given PHP comment line.
func parseParam(line string) (string, []string) {
	lineParts := strings.Split(line, " ")
	if len(lineParts) > 3 {
		paramName := lineParts[3]
		paramTypes := strings.Split(lineParts[2], "|")
		for i, paramType := range paramTypes {
			if renameType, ok := renamedTypes[paramType]; ok {
				paramTypes[i] = renameType
			}
		}
		return paramName, paramTypes
	}
	return "", nil
}
