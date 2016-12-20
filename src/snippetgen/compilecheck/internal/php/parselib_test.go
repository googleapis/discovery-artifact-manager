package php

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"sort"
	"testing"

	"discovery-artifact-manager/snippetgen/compilecheck/internal/filesys"
)

func TestParseFile(t *testing.T) {
	opener := filesys.MapFS{}
	filePath := "testdata/sample-php-client/Appengine/AppsServicesResource.php"
	content, _ := ioutil.ReadFile(filePath)
	opener[filePath] = string(content)
	var actual ParsedLib
	if err := parseFile(&actual, filePath, opener); err != nil {
		t.Fatal(err)
	}
	expected := ParsedLib{
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "delete",
				ParameterName: "$appsId",
			},
			Types: []string{"string", "array"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "delete",
				ParameterName: "$optParams",
			},
			Types: []string{"array"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "delete",
				ParameterName: "$servicesId",
			},
			Types: []string{"string"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "get",
				ParameterName: "$appsId",
			},
			Types: []string{"string"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "get",
				ParameterName: "$servicesId",
			},
			Types: []string{"string"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "get",
				ParameterName: "$optParams",
			},
			Types: []string{"array"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "listAppsServices",
				ParameterName: "$appsId",
			},
			Types: []string{"string"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "listAppsServices",
				ParameterName: "$optParams",
			},
			Types: []string{"array"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "patch",
				ParameterName: "$appsId",
			},
			Types: []string{"string"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "patch",
				ParameterName: "$servicesId",
			},
			Types: []string{"string"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "patch",
				ParameterName: "$postBody",
			},
			Types: []string{"Google_Service"},
		},
		PathTypePair{
			Path: ParameterPath{
				ClassName:     "Google_Service_Appengine_AppsServicesResource",
				MethodName:    "patch",
				ParameterName: "$optParams",
			},
			Types: []string{"array"},
		},
	}
	sort.Sort(expected)
	sort.Sort(actual)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Incorrect parsed library.\n Expected:\n%s\nActual:\n%s\n",
			expected, actual)
	}
}

// Functions used to sort the ParsedLib
func (parsedLib ParsedLib) Len() int {
	return len(parsedLib)
}

func (parsedLib ParsedLib) Swap(i, j int) {
	parsedLib[i], parsedLib[j] = parsedLib[j], parsedLib[i]
}

func (parsedLib ParsedLib) Less(i, j int) bool {
	json1, _ := json.Marshal(parsedLib[i])
	json2, _ := json.Marshal(parsedLib[j])
	return string(json1) < string(json2)
}
