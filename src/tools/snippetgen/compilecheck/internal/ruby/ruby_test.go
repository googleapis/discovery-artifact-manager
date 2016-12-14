package ruby

import (
	"strings"
	"testing"

	"gapi-cmds/src/snippetgen/compilecheck/internal/filesys"
	"gapi-cmds/src/snippetgen/compilecheck/internal/langutil"
)

func TestReadFiles(t *testing.T) {
	files := []struct {
		path, content, initText string
		id                      langutil.MethodID
	}{
		{
			path: "/path/to/pubsub/v1/20151103/pubsub.projects.subscriptions.acknowledge.frag.rb",
			id: langutil.MethodID{
				APIName:      "pubsub",
				APIVersion:   "v1",
				FragmentName: "pubsub.projects.subscriptions.acknowledge",
			},
			content: `# PRE-REQUISITES:
# ---------------
# 1. If not already done, enable the Google Cloud Pub/Sub API and check the quota for your project at
#    https://console.developers.google.com/apis/api/pubsub_component/quotas
# 2. This sample uses Application Default Credentials for Auth. If not already done, install the gcloud CLI from
#    https://cloud.google.com/sdk/ and run 'gcloud beta auth application-default login'
# 3. To install the client library and Application Default Credentials library, run:
#    'gem install google-api-client'
#    'gem install googleauth'
require 'googleauth'
require 'google/apis/pubsub_v1'

Pubsub = Google::Apis::PubsubV1
service = Pubsub::PubsubService.new
service.authorization = Google::Auth.get_application_default(['https://www.googleapis.com/auth/cloud-platform'])

# TODO: Change placeholders below to values for parameters to the 'acknowledge_subscription' method:
# The subscription whose message is being acknowledged.
subscription = ''
resource = Pubsub::AcknowledgeRequest.new

response = service.acknowledge_subscription(subscription, resource)`,
			initText: `# PRE-REQUISITES:
# ---------------
# 1. If not already done, enable the Google Cloud Pub/Sub API and check the quota for your project at
#    https://console.developers.google.com/apis/api/pubsub_component/quotas
# 2. This sample uses Application Default Credentials for Auth. If not already done, install the gcloud CLI from
#    https://cloud.google.com/sdk/ and run 'gcloud beta auth application-default login'
# 3. To install the client library and Application Default Credentials library, run:
#    'gem install google-api-client'
#    'gem install googleauth'
require 'googleauth'
require 'google/apis/pubsub_v1'

Pubsub = Google::Apis::PubsubV1
service = Pubsub::PubsubService.new
service.authorization = Google::Auth.get_application_default(['https://www.googleapis.com/auth/cloud-platform'])

# TODO: Change placeholders below to values for parameters to the 'acknowledge_subscription' method:
# The subscription whose message is being acknowledged.
subscription = ''
resource = Pubsub::AcknowledgeRequest.new

`,
		},
		{
			path: "/path/to/foo/v1/20151103/bar.zip.zap.frag.rb",
			id: langutil.MethodID{
				APIName:      "foo",
				APIVersion:   "v1",
				FragmentName: "bar.zip.zap",
			},
			content: `init code here

  foos = service.fetch_all()
`,
			initText: "init code here",
		},
	}

	opener := filesys.MapFS{}
	var fileNames []string
	for _, f := range files {
		opener[f.path] = f.content
		fileNames = append(fileNames, f.path)
	}
	ctx := checkContext{
		checkFiles: fileNames,
		fs:         opener,
		libDir:     "/libdir",
	}
	if err := readMethodInits(&ctx); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	for _, f := range files {
		switch initText, ok := ctx.MethodInits[f.id]; {
		case !ok:
			t.Errorf("method not found: %q", f.id)
		case strings.TrimSpace(initText) != strings.TrimSpace(f.initText):
			t.Errorf("%s: wrong init text: got %q, want %q", f.id, initText, f.initText)
		}
		delete(ctx.MethodInits, f.id)
	}
	for id := range ctx.MethodInits {
		t.Errorf("method should not exist: %q", id)
	}
}
