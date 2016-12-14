package ruby

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"gapi-cmds/src/snippetgen/compilecheck/internal/langutil"
)

// renameFileName is the location of the client library's Ruby rename file.
var renameFileName = filepath.Join("ruby-client", "google-api-ruby-client-master", "api_names_out.yaml")

// parseLibs parses libraries referenced by `ctx.MethodInits`. It expects the libraries to be under
// `ctx.libDir` and opens the files with `ctx.fs`. It populates `ctx.MethodParamSets`.
func parseLibs(ctx *checkContext) error {
	type libID struct {
		apiName, apiVersion string
	}

	libs := make(map[libID]bool, len(ctx.MethodInits))
	for mid := range ctx.MethodInits {
		libs[libID{
			apiName:    mid.APIName,
			apiVersion: mid.APIVersion,
		}] = true
	}

	ctx.MethodParamSets = make(langutil.MethodParamSets)
	for l := range libs {
		if err := parseLib(ctx, l.apiName, l.apiVersion); err != nil {
			return err
		}
	}
	return nil
}

var (
	// paramRegexp matches docs for parameters like `# @param [String] resource`.
	paramRegexp = regexp.MustCompile(`^#\s+@param\s+\[(.+)\]\s+(\w+)`)

	// methodRegexp matches method definition lines like
	// `def set_topic_iam_policy(resource, set_iam_policy_request_object = nil, fields: nil, quota_user: nil, options: nil, &block)`.
	methodRegexp = regexp.MustCompile(`^def\s+(\w+)\(([\s\S]*?)\)`)

	// genericRegexp matches "generic" parameters in parameter documentation.
	genericRegexp = regexp.MustCompile(`<.*?>`)

	// skipMethodName is the name of a private method generated into all API libraries.
	// We don't perform any check against this method.
	skipMethodName = []byte("apply_command_defaults")
)

// parseLib is a helper of parseLibs, parsing only one file. It updates `ctx.MethodParamSets` with
// the content of the file.
func parseLib(ctx *checkContext, apiName, apiVersion string) error {
	fname := filepath.Join(ctx.libDir, clientLibAPIRoot, apiName+"_"+strings.Replace(apiVersion, ".", "_", -1), "service.rb")
	file, err := ctx.fs.ReadFile(fname)
	if err != nil {
		return err
	}
	var currentParams []langutil.MethodParam

	// Parse the function comment and definition, which looks like:
	//
	// # Sets the access control policy on the specified resource. Replaces any
	// # existing policy.
	// # @param [String] resource
	// #   REQUIRED: The resource for which the policy is being specified. `resource` is
	// #   usually specified as a path, such as `projects/*project*/zones/*zone*/disks/*
	// #   disk*`. The format for the path specified in this value is resource specific
	// #   and is specified in the `setIamPolicy` documentation.
	// # @param [Google::Apis::PubsubV1::SetIamPolicyRequest] set_iam_policy_request_object
	// # @param [String] fields
	// #   Selector specifying which fields to include in a partial response.
	// # @param [String] quota_user
	// #   Available to use for quota purposes for server-side applications. Can be any
	// #   arbitrary string assigned to a user, but should not exceed 40 characters.
	// # @param [Google::Apis::RequestOptions] options
	// #   Request-specific options
	// #
	// # @yield [result, err] Result & error if block supplied
	// # @yieldparam result [Google::Apis::PubsubV1::Policy] parsed result object
	// # @yieldparam err [StandardError] error object if request failed
	// #
	// # @return [Google::Apis::PubsubV1::Policy]
	// #
	// # @raise [Google::Apis::ServerError] An error occurred on the server and the request can be retried
	// # @raise [Google::Apis::ClientError] The request is invalid and should not be retried without modification
	// # @raise [Google::Apis::AuthorizationError] Authorization is required
	// def set_topic_iam_policy(resource, set_iam_policy_request_object = nil, fields: nil, quota_user: nil, options: nil, &block)
	for len(file) > 0 {
		if match := paramRegexp.FindSubmatch(file); len(match) > 0 {
			currentParams = append(currentParams, langutil.MethodParam{
				Name: string(match[2]),
				Type: string(genericRegexp.ReplaceAll(match[1], nil)),
			})
			file = file[len(match[0]):]
		} else if match = methodRegexp.FindSubmatch(file); len(match) > 0 && !bytes.Equal(match[1], skipMethodName) {
			posParams, err := positionalParams(string(match[2]), currentParams)
			if err != nil {
				return err
			}
			friendlyID := langutil.MethodID{
				APIName: apiName, APIVersion: apiVersion, FragmentName: string(match[1]),
			}
			discoID, ok := ctx.methodRename[friendlyID]
			if !ok {
				return fmt.Errorf("rename not found: %v", friendlyID)
			}
			ctx.MethodParamSets[discoID] = posParams
			currentParams = currentParams[:0]
			file = file[len(match[0]):]
		}
		if p := bytes.IndexRune(file, '\n'); p >= 0 {
			file = file[p+1:]
		} else {
			file = nil
		}
		file = bytes.TrimLeftFunc(file, unicode.IsSpace)
	}
	return nil
}

// positionalParams parses a list of of Ruby parameters in `paramList`. Each parameter's type is
// looked up in `paramDefs`. It returns MethodParam's represented by `paramList` and any error.
func positionalParams(paramList string, paramDefs []langutil.MethodParam) ([]langutil.MethodParam, error) {
	var posParams []langutil.MethodParam
LOOP:
	for _, paramSpec := range strings.Split(paramList, ",") {
		paramSpec = strings.TrimSpace(paramSpec)

		// ':' signifies a named parameter and '&' specifies a block. Both must come after all
		// positional parameters.
		if strings.ContainsAny(paramSpec, ":&") {
			break
		}

		var paramName string
		if p := strings.IndexFunc(paramSpec, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_'
		}); p >= 0 {
			paramName = paramSpec[:p]
		} else {
			paramName = paramSpec
		}

		for _, def := range paramDefs {
			if def.Name == paramName {
				// This is a simple heuristic that assumes any param whose type
				// contains "::" is a message, and therefore the request_body.
				if strings.Contains(def.Type, "::") {
					def.Name = "request_body"
				}
				posParams = append(posParams, def)
				continue LOOP
			}
		}
		return nil, fmt.Errorf("param used but not defined: %q", paramName)
	}
	return posParams, nil
}

// parseNameMap parses a Ruby rename file, creating a map from each method's
// user-friendly name to its discovery name. All non-method renames are dropped.
//
// A Ruby rename file is a YAML file where each line is a mapping of a discovery name (used
// in Discovery docs) to a user-friendly name (used in the Ruby client library). For an example, see
// TestParseNameMap.
func parseNameMap(ctx *checkContext) error {
	rd, err := ctx.fs.Open(filepath.Join(ctx.libDir, renameFileName))
	if err != nil {
		return err
	}
	defer rd.Close()

	slashSep := []byte("/")

	ctx.methodRename = make(map[langutil.MethodID]langutil.MethodID)
	sc := bufio.NewScanner(rd)
	for sc.Scan() {
		// Format:
		//   "/admin:directory_v1/directory.mobiledevices.list": list_mobile_devices
		// Only methods have exactly two slashes.
		if bytes.Count(sc.Bytes(), slashSep) != 2 {
			continue
		}
		line := sc.Text()
		if p := strings.LastIndex(line, ":"); p >= 0 {
			friendlyName := strings.TrimSpace(line[p+1:])
			discoID, err := parseRenameKey(line[:p])
			if err != nil {
				return err
			}
			friendlyID := discoID
			friendlyID.FragmentName = friendlyName
			ctx.methodRename[friendlyID] = discoID
		}
	}
	return sc.Err()
}

// parseRenameKey parses a discovery name, as found in Ruby rename files, into MethodID. It returns
// the MethodID and any error encountered.
func parseRenameKey(s string) (langutil.MethodID, error) {
	// Format: "/admin:directory_v1/directory.mobiledevices.list"
	s = strings.Trim(s, `"/`)
	p := strings.IndexRune(s, '/')
	if p < 0 {
		return langutil.MethodID{}, fmt.Errorf("parseRenameKey: cannot find API/method separator: %s", s)
	}
	api := s[:p]
	methodName := s[p+1:]
	if p = strings.IndexRune(api, ':'); p < 0 {
		return langutil.MethodID{}, fmt.Errorf("parseRenameKey: cannot find API name/version separator: %s", s)
	}
	return langutil.MethodID{
		APIName:      api[:p],
		APIVersion:   api[p+1:],
		FragmentName: methodName,
	}, nil
}
