// Package php implements compilecheck for PHP.
//
// The snippet generation process derives its information from the discovery doc that should work
// with the client library. We perform a "compile check" by extracting the type information from
// the client library documentation, and making sure it matches the types we deduced.
package php

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"discovery-artifact-manager/common/environment"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/filesys"
)

// Sample represents a parsed sample snippet.
type Sample struct {
	Service         string
	InitLines       []string
	MethodSignature MethodSignature
}

// MethodSignature contains information of the method from the sample snippet.
type MethodSignature struct {
	Identifier string
	Path       string
	Method     string
	Params     []string
}

// Check sets up the environment, parses information from the PHP
// client library and generates the PHP files that are used for the
// compile check.
func Check(files []string, libDir, testDir string) (string, error) {
	if err := setup(testDir); err != nil {
		return "", err
	}

	var samples []Sample
	for _, file := range files {
		var sample, err = readFile(file, filesys.OS{})
		samples = append(samples, sample)
		if err != nil {
			return "", err
		}
	}

	// For the time being, ignore the passed-in libDir and
	// overwrite it with a path to this subrepo within this
	// repository
	//
	// TODO(vchudnov-g): Use the passed in libDir once the default
	// value for --lib has been changed in compilecheck.go
	dartmanDir, err := environment.RepoRoot()
	if err != nil {
		return "", err
	}
	libDir = fmt.Sprintf("%s/clients/php/google-api-php-client-services", dartmanDir)

	parsedLib, err := parseLibs(libDir, filesys.OS{})
	if err != nil {
		return "", err
	}
	filePath, _, err := render(testDir, filesys.OS{}, samples, parsedLib)

	ioutil.WriteFile(filepath.Join(testDir, "composer.json"), []byte(fmt.Sprintf(`{
    "repositories": [
        {
            "type": "path",
            "url": "%s",
            "options": {
              "symlink": true
            }
        }
    ],
    "require": {
        "google/apiclient-services": "*"
    }
}`, libDir)), os.ModePerm)

	return fmt.Sprintf("(cd %s;composer require google/apiclient;"+
		"composer require;php %s)\n",
		testDir, filePath), err
}

// setup sets up the enviroment for compile check.
func setup(testDir string) error {
	if err := os.RemoveAll(testDir); err != nil {
		return err
	}
	if err := os.MkdirAll(testDir, 0750); err != nil {
		return err
	}
	return nil
}

// Patterns to identify various types of lines in the snippets.
const (
	clientLine     = "$client ="
	serviceLine    = "$service ="
	methodCallLine = "$service->"
	commentLine    = "//"
	endTagLine     = "?>"
	serviceVarName = "service"
	clientVarName  = "client"
)

// readFile reads a single code sample file `fname` using `opener`
// Returns a Sample which presents the sample PHP code and the error (if any).
func readFile(fname string, opener filesys.Opener) (Sample, error) {
	file, err := opener.Open(fname)
	if err != nil {
		return Sample{}, fmt.Errorf("error in readFile(%q): %q", fname, err)
	}
	defer file.Close()

	var initLines []string
	var inInit bool
	var paramNames, path []string
	var requestBodyParamNameMap = make(map[string]bool)
	var service, method string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Pretend any occurrence of requestBody is actually postBody.
		// Compilecheck does 1:1 name comparisons, so leaving the name
		// as-is won't work.
		line = strings.Replace(line, "requestBody", "postBody", -1)
		lineParts := splitPHPLine(line)
		if strings.HasPrefix(line, commentLine) || len(line) == 0 || line == endTagLine {
			// Skip comments, empty lines, and closing tags.
			continue
		}
		if strings.Contains(line, clientLine) {
			inInit = true
		}
		if strings.Contains(line, serviceLine) {
			service = parseServiceName(lineParts)
		}
		if line == "do {" || strings.HasPrefix(line, "} while") {
			continue
		}
		if inInit && strings.Contains(line, methodCallLine) {
			inInit = false
			path, method = parseMethodSignature(lineParts)
		}
		if strings.HasPrefix(line, "$postBody->") {
			i := strings.Index(line[1:], "$")
			j := strings.LastIndex(line, ")")
			paramName := line[i+2 : j]
			requestBodyParamNameMap[paramName] = true
		}
		if inInit {
			initLines = append(initLines, line)
			if paramName := parseParamName(lineParts); paramName != "" &&
				paramName != serviceVarName && paramName != clientVarName {
				// Skip optional parameters and assignments to the `optParams` map.
				if paramName == "optParams" || strings.HasPrefix(paramName, "optParams[") {
					continue
				}

				paramNames = append(paramNames, paramName)
			}
		}
	}

	// Remake the `paramNames` array to exclude variables that are only part of the request body.
	var tmpParamNames []string
	for i := 0; i < len(paramNames); i++ {
		if !requestBodyParamNameMap[paramNames[i]] {
			tmpParamNames = append(tmpParamNames, paramNames[i])
		}
	}
	paramNames = tmpParamNames

	methodSignature := MethodSignature{
		Identifier: service + strings.Join(path, "") + method,
		Path:       strings.Join(path, "->"),
		Method:     method,
		Params:     paramNames,
	}

	sample := Sample{
		Service:         service,
		MethodSignature: methodSignature,
		InitLines:       initLines,
	}
	return sample, scanner.Err()
}

// parseParamName parses the parameter name from the line.
func parseParamName(lineParts []string) string {
	if len(lineParts) > 0 && !strings.Contains(lineParts[0], "->") {
		return lineParts[0]
	}
	return ""
}

// parseMethodSignature parses the method information from the line.
func parseMethodSignature(lineParts []string) ([]string, string) {
	if lineParts[0] == "response" {
		lineParts = lineParts[1:]
	}
	if len(lineParts) > 0 {
		methodCallString := lineParts[0]
		methodPath := strings.Split(methodCallString, "->")
		return methodPath[0 : len(methodPath)-1],
			methodPath[len(methodPath)-1]
	}
	return nil, ""
}

// parseServiceName parses the service name from the line.
func parseServiceName(lineParts []string) string {
	if len(lineParts) > 2 {
		return lineParts[2]
	}
	return ""
}

// splitPHPLine is a helper method that splits a PHP line with some common delimiters.
func splitPHPLine(line string) []string {
	return strings.FieldsFunc(line, func(r rune) bool {
		return strings.ContainsRune("() $=;", r)
	})
}
