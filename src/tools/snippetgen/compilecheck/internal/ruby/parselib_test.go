package ruby

import (
	"reflect"
	"testing"

	"gapi-cmds/src/snippetgen/compilecheck/internal/filesys"
	"gapi-cmds/src/snippetgen/compilecheck/internal/langutil"
)

func TestParseLibs(t *testing.T) {
	ctx := checkContext{
		MethodInits: map[langutil.MethodID]string{
			// parseLibs does not use FragmentName or init code.
			{APIName: "myAPI", APIVersion: "v1", FragmentName: ""}:    "",
			{APIName: "otherAPI", APIVersion: "v2", FragmentName: ""}: "",
		},
		methodRename: map[langutil.MethodID]langutil.MethodID{
			{APIName: "myAPI", APIVersion: "v1", FragmentName: "my_method"}:               {APIName: "myAPI", APIVersion: "v1", FragmentName: "my.method"},
			{APIName: "otherAPI", APIVersion: "v2", FragmentName: "set_topic_iam_policy"}: {APIName: "otherAPI", APIVersion: "v2", FragmentName: "set.topic.iam.policy"},
		},
		libDir: "path/to/lib",
		fs: filesys.MapFS{
			"path/to/lib/ruby-client/google-api-ruby-client-master/generated/google/apis/myAPI_v1/service.rb": `
			# @param [String] foo
			#	Description of foo
			# @param [Fixnum] bar
			#	Description of bar
			def my_method(foo, bar = 0)
			`,

			"path/to/lib/ruby-client/google-api-ruby-client-master/generated/google/apis/otherAPI_v2/service.rb": `
			# Sets the access control policy on the specified resource. Replaces any
			# existing policy.
			# @param [String] resource
			#	 REQUIRED: The resource for which the policy is being specified. resource is
			#	 usually specified as a path, such as projects/*project*/zones/*zone*/disks/*
			#	 disk*. The format for the path specified in this value is resource specific
			#	 and is specified in the setIamPolicy documentation.
			# @param [Google::Apis::PubsubV1::SetIamPolicyRequest] set_iam_policy_request_object
			# @param [String] fields
			#	 Selector specifying which fields to include in a partial response.
			# @param [String] quota_user
			#	 Available to use for quota purposes for server-side applications. Can be any
			#	 arbitrary string assigned to a user, but should not exceed 40 characters.
			# @param [Google::Apis::RequestOptions] options
			#	 Request-specific options
			#
			# @yield [result, err] Result & error if block supplied
			# @yieldparam result [Google::Apis::PubsubV1::Policy] parsed result object
			# @yieldparam err [StandardError] error object if request failed
			#
			# @return [Google::Apis::PubsubV1::Policy]
			#
			# @raise [Google::Apis::ServerError] An error occurred on the server and the request can be retried
			# @raise [Google::Apis::ClientError] The request is invalid and should not be retried without modification
			# @raise [Google::Apis::AuthorizationError] Authorization is required
			def set_topic_iam_policy(resource, set_iam_policy_request_object = nil, fields: nil, quota_user: nil, options: nil, &block)`,
		},
	}
	if err := parseLibs(&ctx); err != nil {
		t.Fatal(err)
	}

	want := langutil.MethodParamSets{
		{APIName: "myAPI", APIVersion: "v1", FragmentName: "my.method"}: {
			{Name: "foo", Type: "String"},
			{Name: "bar", Type: "Fixnum"},
		},
		{APIName: "otherAPI", APIVersion: "v2", FragmentName: "set.topic.iam.policy"}: {
			{Name: "resource", Type: "String"},
			{Name: "request_body", Type: "Google::Apis::PubsubV1::SetIamPolicyRequest"},
		},
	}
	if got := ctx.MethodParamSets; !reflect.DeepEqual(got, want) {
		t.Errorf("wrong MethodParamSets, got:\n%v\nwant:\n%v", got, want)
	}
}

func TestParseNameMap(t *testing.T) {
	ctx := checkContext{
		libDir: "path/to/lib",
		fs: filesys.MapFS{
			"path/to/lib/ruby-client/google-api-ruby-client-master/api_names_out.yaml": `
---
"/admin:directory_v1/directory.mobiledevices.list": list_mobile_devices
"/foo:v1/foo.bar.mymethod": foo_bar_method
`,
		},
	}
	if err := parseNameMap(&ctx); err != nil {
		t.Fatal(err)
	}

	want := map[langutil.MethodID]langutil.MethodID{
		{"admin", "directory_v1", "list_mobile_devices"}: {"admin", "directory_v1", "directory.mobiledevices.list"},
		{"foo", "v1", "foo_bar_method"}:                  {"foo", "v1", "foo.bar.mymethod"},
	}
	if got := ctx.methodRename; !reflect.DeepEqual(got, want) {
		t.Errorf("wrong methodRename, got:\n%v\nwant:\n%v", got, want)
	}
}
