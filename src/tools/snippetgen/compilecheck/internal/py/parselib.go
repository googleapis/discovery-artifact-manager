package py

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gapi-cmds/src/common/parsehtml"
	"gapi-cmds/src/snippetgen/compilecheck/internal/langutil"

	"golang.org/x/net/html"
)

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
	// isMethodDiv identifies HTML div elements for detailed method docs.
	isMethodDiv = parsehtml.NodeIsAll(parsehtml.HasElementName("div"), parsehtml.HasClass("method"))

	// isSignature identifies HTML code elements for method call signatures.
	isSignature = parsehtml.NodeIsAll(parsehtml.HasElementName("code"), parsehtml.HasClass("details"))

	// isDescription identifies HTML preformatted-text elements for method descriptions.
	isDescription = parsehtml.HasElementName("pre")

	// signaturePattern matches method call signatures like:
	// 	get(jobId, reportId, onBehalfOfContentOwner=None, x__xgafv=None)
	signaturePattern = regexp.MustCompile(`(\w+)\((.*)\)`)

	// paramsHeadPattern matches argument description section head of method detail like:
	//
	// 	Args:
	// Uses only one leading line break due to bug in html.Parse affecting method description
	// blocks with no leading description before parameter listing.
	paramsHeadPattern = "\nArgs:\n"

	// returnsHeadPattern matches returns description section head of method detail like:
	//
	// 	Returns:
	returnsHeadPattern = "\n\nReturns:\n"

	// paramPattern matches argument name/type descriptions like:
	// 	jobId: string, ... (repeated)
	paramPattern = regexp.MustCompile(`(?m)^  (\w+): (\w+),.*?((?:\(repeated\))?)$`)

	// uppercasePattern matches any sequence of 1 or more uppercase letters. It's used to
	// convert camelCase identifiers to snake_case.
	uppercasePattern = regexp.MustCompile(`([A-Z]+)`)
)

// parseLib parses PyDoc files for one API version, updating `ctx.MethodParamSets`.
func parseLib(ctx *checkContext, apiName, apiVersion string) error {
	// ignore top-level PyDocs ("apiName_apiVersion.html"), which contain only non-API helper methods
	filenames, err := filepath.Glob(filepath.Join(ctx.libDir, clientLibAPIRoot,
		apiName+"_"+strings.Replace(apiVersion, ".", "_", -1)+".*.html"))
	if err != nil {
		return err
	}
	for _, filename := range filenames {
		if err = parseFile(ctx, apiName, apiVersion, filename); err != nil {
			return err
		}
	}
	return nil
}

// parseFile parses an individual PyDoc file, corresponding to one resource kind in one API version,
// updating `ctx.MethodParamSets`.
func parseFile(ctx *checkContext, apiName, apiVersion, filename string) error {
	file, err := ctx.fs.ReadFile(filename)
	if err != nil {
		return err
	}
	// Parse the PyDoc HTML, with the following substructure (roughly: * denotes
	// repetition; [] denotes optional; paramType is one of {boolean, integer, number,
	// object, string}):
	//
	// <html>
	// 	<body>
	// 		<div class="method">
	//			<code class="details" ...>methodName(paramName, *[paramName=defaultValue, ]*)</code>
	//			<pre>
	//
	//				Args:
	//				  paramName: paramType, ...
	//				  *
	//
	//				[Returns:]
	//			</pre>
	//		</div>
	//		*
	//	</body>
	// </html>
	pydoc, err := html.Parse(bytes.NewReader(file))
	if err != nil {
		return err
	}
	document := parsehtml.Node{pydoc}.FindChildNode(parsehtml.HasElementName("html"))
	if document.Node == nil {
		return fmt.Errorf("PyDoc missing HTML: %v", filename)
	}
	body := document.FindChildNode(parsehtml.HasElementName("body"))
	if body.Node == nil {
		return fmt.Errorf("PyDoc HTML missing body: %v", filename)
	}
	// parse method ID prefix
	prefix := parsehtml.InBetween(filename, apiName+"_"+strings.Replace(apiVersion, ".", "_", -1)+".", "html")
	// parse method specifications
	err = body.OnEachChildNode(isMethodDiv, func(methodDiv parsehtml.Node) error {
		// parse method signature
		signature := methodDiv.FindChildNode(isSignature)
		if signature.Node == nil {
			return fmt.Errorf("method signature not found: %v", methodDiv)
		}
		signatureText, err := signature.Text()
		if err != nil {
			return err
		}
		signatureMethodParams := signaturePattern.FindStringSubmatch(signatureText)
		if signatureMethodParams == nil {
			return fmt.Errorf("method signature has unexpected form: %v", signatureText)
		}
		methodName := string(signatureMethodParams[1])
		paramList := string(signatureMethodParams[2])
		if strings.HasSuffix(methodName, "_next") {
			// ignore non-API helper methods like `list_next`
			return nil
		}
		methodID := langutil.MethodID{
			APIName:      apiName,
			APIVersion:   apiVersion,
			FragmentName: prefix + methodName,
		}
		description, err := signature.FindNextNode(isDescription).Text()
		if err != nil {
			return err
		}
		paramsSection := parsehtml.InBetween(description, paramsHeadPattern, returnsHeadPattern)
		paramNameTypes := paramPattern.FindAllStringSubmatch(paramsSection, -1)
		paramDefs := make([]langutil.MethodParam, len(paramNameTypes))
		for i, paramNameType := range paramNameTypes {
			// If (repeated) is captured, then substitute the type as list.
			if paramNameType[3] != "" {
				paramNameType[2] = "list"
			}
			paramDefs[i] = langutil.MethodParam{
				Name: toSnakeCase(string(paramNameType[1])),
				Type: string(paramNameType[2]),
			}
		}
		// associate parameter descriptions to required parameters
		reqParams, err := requiredParams(paramList, paramDefs)
		if err != nil {
			return err
		}
		ctx.MethodParamSets[methodID] = reqParams
		return nil
	})
	return err
}

// requiredParams parses parameters in the given Python method arguments signature `paramList`,
// returning a list of MethodParams found by lookup in the `paramDefs` list.
func requiredParams(paramList string, paramDefs []langutil.MethodParam) ([]langutil.MethodParam, error) {
	var reqParams []langutil.MethodParam
	if paramList == "" {
		return reqParams, nil
	}
ParamList:
	for _, paramName := range strings.Split(paramList, ",") {
		paramName = toSnakeCase(strings.TrimSpace(paramName))
		// '=' signifies an optional parameter, which must come after required positional
		// parameters: currently, no variadic methods appear in client libraries, nor
		// default values other than `None`; more complicated method signatures may require
		// more sophisticated parsing
		if strings.ContainsAny(paramName, "=") {
			break
		}

		for _, def := range paramDefs {
			if def.Name == paramName {
				reqParams = append(reqParams, def)
				continue ParamList
			}
		}
		return nil, fmt.Errorf("parameter used but not defined: %q", paramName)
	}
	return reqParams, nil
}

// toSnakeCase converts ident from lowerCamel to snake_case format.
func toSnakeCase(ident string) string {
	ident = uppercasePattern.ReplaceAllString(ident, `_${1}`)
	return strings.ToLower(ident)
}
