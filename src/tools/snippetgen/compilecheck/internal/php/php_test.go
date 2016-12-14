package php

import (
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"discovery-artifact-manager/tools/snippetgen/compilecheck/internal/filesys"
)

func TestReadFileNotExist(t *testing.T) {
	opener := filesys.MapFS{}
	const fname = "myservice/v1/1234/somethingNotExist.php"
	if _, err := readFile(fname, opener); err == nil {
		t.Errorf("Expected error since file %q does not exist", fname)
	}
}

func TestReadFile(t *testing.T) {
	opener := filesys.MapFS{}
	filePath := "testdata/sample.frag.php"
	content, _ := ioutil.ReadFile(filePath)

	if !strings.Contains(string(content), "$requestBody") {
		t.Errorf("Sample fragment doesn't test rename of $requestBody")
	}

	opener[filePath] = string(content)
	sample, err := readFile(filePath, opener)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if sample.Service != "Google_Service_Appengine" {
		t.Errorf("Incorrect service name.\nExpected: Google_Service_Appengine\nActual: %s\n",
			sample.Service)
	}
	expectedInitLines := []string{
		"$client = new Google_Client();",
		"$client->setApplicationName('Client Sample Application');",
		"$client->useApplicationDefaultCredentials();",
		"$client->addScope('https://www.googleapis.com/auth/cloud-platform');",
		"$service = new Google_Service_Appengine($client);",
		"$appsId = '';",
		"$postBody = object();",
	}
	if !reflect.DeepEqual(expectedInitLines, sample.InitLines) {
		t.Errorf("Incorrect init lines.\nExpected: %s\nActual: %s\n", expectedInitLines,
			sample.InitLines)
	}

	expectedSignature := MethodSignature{
		Identifier: "Google_Service_Appengineserviceappsget",
		Path:       "service->apps",
		Method:     "get",
		Params:     []string{"appsId", "postBody"},
	}
	if !reflect.DeepEqual(expectedSignature, sample.MethodSignature) {
		t.Errorf("Incorrect method signiture.\nExpected: %s\nActual: %s\n", expectedSignature,
			sample.MethodSignature)
	}
}
