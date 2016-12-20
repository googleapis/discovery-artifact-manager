package nodejs

import (
	"strings"
	"testing"

	"discovery-artifact-manager/snippetgen/compilecheck/internal/filesys"
	"discovery-artifact-manager/snippetgen/compilecheck/internal/langutil"
)

func TestReadFilesBadName(t *testing.T) {
	opener := filesys.MapFS{}
	const fname = "foo"
	if _, err := readFiles([]string{fname}, opener); err == nil {
		t.Errorf("expected error since file %q is not in proper format", fname)
	}
}

func TestReadFilesNotExist(t *testing.T) {
	opener := filesys.MapFS{}
	const fname = "myservice/v1/1234/myservice.mymethod.frag.njs"
	if _, err := readFiles([]string{fname}, opener); err == nil {
		t.Errorf("expected error since file %q does not exist", fname)
	}
}

func TestReadFiles(t *testing.T) {
	files := []struct {
		path, content, initText string
		id                      langutil.MethodID
	}{
		{
			path: "/path/to/appengine/v1beta4/1234/appengine.apps.get.frag.njs",
			id:   langutil.MethodID{APIName: "appengine", APIVersion: "v1beta4", FragmentName: "appengine.apps.get"},
			initText: `
var request = {
// TODO: Change placeholders below to values for parameters to the 'get' method:

// Part of name. Name of the application to get. For example: "apps/myapp".
appsId: "",
// Auth client
auth: authClient
};`,
			content: `
var google = require('googleapis');
var GoogleAuth = require('google-auth-library');

var authFactory = new GoogleAuth();
var appengine = google.appengine('v1beta4');

authFactory.getApplicationDefault(function(err, authClient) {
  if (err) {
    console.log('Authentication failed because of ', err);
    return;
  }
  if (authClient.createScopedRequired && authClient.createScopedRequired()) {
    var scopes = ['https://www.googleapis.com/auth/cloud-platform'];
    authClient = authClient.createScoped(scopes);
  }

  var request = {
    // TODO: Change placeholders below to values for parameters to the 'get' method:

    // Part of name. Name of the application to get. For example: "apps/myapp".
    appsId: "",
    // Auth client
    auth: authClient
  };

  appengine.apps.get(request, function(err, result) {
    if (err) {
      console.log(err);
    } else {
      console.log(result);
    }
  });
});
`,
		},
		{
			path: "/path/to/foo/v1/1234/foo.bar.frag.njs",
			id:   langutil.MethodID{APIName: "foo", APIVersion: "v1", FragmentName: "foo.bar"},
			initText: `
var request = {
appsId: "",
resource: {},
auth: authClient
};
`,
			content: `
var request = {
appsId: "",
resource: {},
auth: authClient
};
`,
		},
	}

	opener := filesys.MapFS{}
	var fileNames []string
	for _, f := range files {
		opener[f.path] = f.content
		fileNames = append(fileNames, f.path)
	}

	sampleMethods, err := readFiles(fileNames, opener)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	for _, f := range files {
		switch initText, ok := sampleMethods[f.id]; {
		case !ok:
			t.Errorf("method not found: %q", f.id)
		case strings.TrimSpace(initText) != strings.TrimSpace(f.initText):
			t.Errorf("%s: wrong init text: got %q, want %q", f.id, initText, f.initText)
		}
		delete(sampleMethods, f.id)
	}
	for id := range sampleMethods {
		t.Errorf("method should not exist: %q", id)
	}
}
