package py

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"gapi-cmds/src/snippetgen/compilecheck/internal/filesys"
	"gapi-cmds/src/snippetgen/compilecheck/internal/langutil"
)

func TestParseFile(t *testing.T) {
	ctx := checkContext{
		MethodParamSets: make(langutil.MethodParamSets),
		MethodInits: map[langutil.MethodID]string{
			// parseLibs does not use FragmentName or init code.
			{APIName: "pubsub", APIVersion: "v1", FragmentName: ""}: "",
		},
		libDir: "path/to/lib",
		fs: filesys.MapFS{
			"path/to/lib/python-client/google-api-python-client-master/docs/dyn/pubsub_v1.projects.subscriptions.html": `
<html><body>
<style>
</style>

<h1><a href="pubsub_v1.html">Google Cloud Pub/Sub API</a> . <a href="pubsub_v1.projects.html">projects</a> . <a href="pubsub_v1.projects.subscriptions.html">subscriptions</a></h1>
<h2>Instance Methods</h2>
<h3>Method Details</h3>
<div class="method">
    <code class="details" id="create">create(name, listEx, body, x__xgafv=None)</code>
  <pre>Creates a subscription to a given topic for a given subscriber. If the subscription already exists, returns ` + "`" + `ALREADY_EXISTS` + "`" + `. If the corresponding topic doesn't exist, returns ` + "`" + `NOT_FOUND` + "`" + `. If the name is not provided in the request, the server will assign a random name for this subscription on the same project as the topic.

Args:
  name: string, The name of the subscription. It must have the format ` + "`" + `"projects/{project}/subscriptions/{subscription}"` + "`" + `. ` + "`" + `{subscription}` + "`" + ` must start with a letter, and contain only letters (` + "`" + `[A-Za-z]` + "`" + `), numbers (` + "`" + `[0-9]` + "`" + `), dashes (` + "`" + `-` + "`" + `), underscores (` + "`" + `_` + "`" + `), periods (` + "`" + `.` + "`" + `), tildes (` + "`" + `~` + "`" + `), plus (` + "`" + `+` + "`" + `) or percent signs (` + "`" + `%` + "`" + `). It must be between 3 and 255 characters in length, and it must not start with ` + "`" + `"goog"` + "`" + `. (required)
  listEx: string, Lorem ipsum dolor. (required) (repeated)
  body: object, The request body. (required)
    The object takes the form of:

{ # A subscription resource.
  "topic": "A String", # The name of the topic from which this subscription is receiving messages. The value of this field will be ` + "`" + `_deleted-topic_` + "`" + ` if the topic has been deleted.
  "ackDeadlineSeconds": 42, # This value is the maximum time after a subscriber receives a message before the subscriber should acknowledge the message. After message delivery but before the ack deadline expires and before the message is acknowledged, it is an outstanding message and will not be delivered again during that time (on a best-effort basis). For pull delivery this value is used as the initial value for the ack deadline. To override this value for a given message, call ` + "`" + `ModifyAckDeadline` + "`" + ` with the corresponding ` + "`" + `ack_id` + "`" + `. For push delivery, this value is also used to set the request timeout for the call to the push endpoint. If the subscriber never acknowledges the message, the Pub/Sub system will eventually redeliver the message. If this parameter is not set, the default value of 10 seconds is used.
  "pushConfig": { # Configuration for a push delivery endpoint. # If push delivery is used with this subscription, this field is used to configure it. An empty ` + "`" + `pushConfig` + "`" + ` signifies that the subscriber will pull and ack messages using API methods.
    "attributes": { # Endpoint configuration attributes. Every endpoint has a set of API supported attributes that can be used to control different aspects of the message delivery. The currently supported attribute is ` + "`" + `x-goog-version` + "`" + `, which you can use to change the format of the push message. This attribute indicates the version of the data expected by the endpoint. This controls the shape of the envelope (i.e. its fields and metadata). The endpoint version is based on the version of the Pub/Sub API. If not present during the ` + "`" + `CreateSubscription` + "`" + ` call, it will default to the version of the API used to make such call. If not present during a ` + "`" + `ModifyPushConfig` + "`" + ` call, its value will not be changed. ` + "`" + `GetSubscription` + "`" + ` calls will always return a valid version, even if the subscription was created without this attribute. The possible values for this attribute are: * ` + "`" + `v1beta1` + "`" + `: uses the push format defined in the v1beta1 Pub/Sub API. * ` + "`" + `v1` + "`" + ` or ` + "`" + `v1beta2` + "`" + `: uses the push format defined in the v1 Pub/Sub API.
      "a_key": "A String",
    },
    "pushEndpoint": "A String", # A URL locating the endpoint to which messages should be pushed. For example, a Webhook endpoint might use "https://example.com/push".
  },
  "name": "A String", # The name of the subscription. It must have the format ` + "`" + `"projects/{project}/subscriptions/{subscription}"` + "`" + `. ` + "`" + `{subscription}` + "`" + ` must start with a letter, and contain only letters (` + "`" + `[A-Za-z]` + "`" + `), numbers (` + "`" + `[0-9]` + "`" + `), dashes (` + "`" + `-` + "`" + `), underscores (` + "`" + `_` + "`" + `), periods (` + "`" + `.` + "`" + `), tildes (` + "`" + `~` + "`" + `), plus (` + "`" + `+` + "`" + `) or percent signs (` + "`" + `%` + "`" + `). It must be between 3 and 255 characters in length, and it must not start with ` + "`" + `"goog"` + "`" + `.
}

  x__xgafv: string, V1 error format.

Returns:
  An object of the form:

    { # A subscription resource.
    "topic": "A String", # The name of the topic from which this subscription is receiving messages. The value of this field will be ` + "`" + `_deleted-topic_` + "`" + ` if the topic has been deleted.
    "ackDeadlineSeconds": 42, # This value is the maximum time after a subscriber receives a message before the subscriber should acknowledge the message. After message delivery but before the ack deadline expires and before the message is acknowledged, it is an outstanding message and will not be delivered again during that time (on a best-effort basis). For pull delivery this value is used as the initial value for the ack deadline. To override this value for a given message, call ` + "`" + `ModifyAckDeadline` + "`" + ` with the corresponding ` + "`" + `ack_id` + "`" + `. For push delivery, this value is also used to set the request timeout for the call to the push endpoint. If the subscriber never acknowledges the message, the Pub/Sub system will eventually redeliver the message. If this parameter is not set, the default value of 10 seconds is used.
    "pushConfig": { # Configuration for a push delivery endpoint. # If push delivery is used with this subscription, this field is used to configure it. An empty ` + "`" + `pushConfig` + "`" + ` signifies that the subscriber will pull and ack messages using API methods.
      "attributes": { # Endpoint configuration attributes. Every endpoint has a set of API supported attributes that can be used to control different aspects of the message delivery. The currently supported attribute is ` + "`" + `x-goog-version` + "`" + `, which you can use to change the format of the push message. This attribute indicates the version of the data expected by the endpoint. This controls the shape of the envelope (i.e. its fields and metadata). The endpoint version is based on the version of the Pub/Sub API. If not present during the ` + "`" + `CreateSubscription` + "`" + ` call, it will default to the version of the API used to make such call. If not present during a ` + "`" + `ModifyPushConfig` + "`" + ` call, its value will not be changed. ` + "`" + `GetSubscription` + "`" + ` calls will always return a valid version, even if the subscription was created without this attribute. The possible values for this attribute are: * ` + "`" + `v1beta1` + "`" + `: uses the push format defined in the v1beta1 Pub/Sub API. * ` + "`" + `v1` + "`" + ` or ` + "`" + `v1beta2` + "`" + `: uses the push format defined in the v1 Pub/Sub API.
        "a_key": "A String",
      },
      "pushEndpoint": "A String", # A URL locating the endpoint to which messages should be pushed. For example, a Webhook endpoint might use "https://example.com/push".
    },
    "name": "A String", # The name of the subscription. It must have the format ` + "`" + `"projects/{project}/subscriptions/{subscription}"` + "`" + `. ` + "`" + `{subscription}` + "`" + ` must start with a letter, and contain only letters (` + "`" + `[A-Za-z]` + "`" + `), numbers (` + "`" + `[0-9]` + "`" + `), dashes (` + "`" + `-` + "`" + `), underscores (` + "`" + `_` + "`" + `), periods (` + "`" + `.` + "`" + `), tildes (` + "`" + `~` + "`" + `), plus (` + "`" + `+` + "`" + `) or percent signs (` + "`" + `%` + "`" + `). It must be between 3 and 255 characters in length, and it must not start with ` + "`" + `"goog"` + "`" + `.
  }</pre>
</div>

<div class="method">
    <code class="details" id="list">list(projectId, pageSize=None, pageToken=None, x__xgafv=None)</code>
  <pre>Lists matching subscriptions.

Args:
  projectId: string, The name of the cloud project that subscriptions belong to. (required)
  pageSize: integer, Maximum number of subscriptions to return.
  pageToken: string, The value returned by the last ` + "`" + `ListSubscriptionsResponse` + "`" + `; indicates that this is a continuation of a prior ` + "`" + `ListSubscriptions` + "`" + ` call, and that the system should return the next page of data.
  x__xgafv: string, V1 error format.

Returns:
  An object of the form:

    { # Response for the ` + "`" + `ListSubscriptions` + "`" + ` method.
    "nextPageToken": "A String", # If not empty, indicates that there may be more subscriptions that match the request; this value should be passed in a new ` + "`" + `ListSubscriptionsRequest` + "`" + ` to get more subscriptions.
    "subscriptions": [ # The subscriptions that match the request.
      { # A subscription resource.
        "topic": "A String", # The name of the topic from which this subscription is receiving messages. The value of this field will be ` + "`" + `_deleted-topic_` + "`" + ` if the topic has been deleted.
        "ackDeadlineSeconds": 42, # This value is the maximum time after a subscriber receives a message before the subscriber should acknowledge the message. After message delivery but before the ack deadline expires and before the message is acknowledged, it is an outstanding message and will not be delivered again during that time (on a best-effort basis). For pull delivery this value is used as the initial value for the ack deadline. To override this value for a given message, call ` + "`" + `ModifyAckDeadline` + "`" + ` with the corresponding ` + "`" + `ack_id` + "`" + `. For push delivery, this value is also used to set the request timeout for the call to the push endpoint. If the subscriber never acknowledges the message, the Pub/Sub system will eventually redeliver the message. If this parameter is not set, the default value of 10 seconds is used.
        "pushConfig": { # Configuration for a push delivery endpoint. # If push delivery is used with this subscription, this field is used to configure it. An empty ` + "`" + `pushConfig` + "`" + ` signifies that the subscriber will pull and ack messages using API methods.
          "attributes": { # Endpoint configuration attributes. Every endpoint has a set of API supported attributes that can be used to control different aspects of the message delivery. The currently supported attribute is ` + "`" + `x-goog-version` + "`" + `, which you can use to change the format of the push message. This attribute indicates the version of the data expected by the endpoint. This controls the shape of the envelope (i.e. its fields and metadata). The endpoint version is based on the version of the Pub/Sub API. If not present during the ` + "`" + `CreateSubscription` + "`" + ` call, it will default to the version of the API used to make such call. If not present during a ` + "`" + `ModifyPushConfig` + "`" + ` call, its value will not be changed. ` + "`" + `GetSubscription` + "`" + ` calls will always return a valid version, even if the subscription was created without this attribute. The possible values for this attribute are: * ` + "`" + `v1beta1` + "`" + `: uses the push format defined in the v1beta1 Pub/Sub API. * ` + "`" + `v1` + "`" + ` or ` + "`" + `v1beta2` + "`" + `: uses the push format defined in the v1 Pub/Sub API.
            "a_key": "A String",
          },
          "pushEndpoint": "A String", # A URL locating the endpoint to which messages should be pushed. For example, a Webhook endpoint might use "https://example.com/push".
        },
        "name": "A String", # The name of the subscription. It must have the format ` + "`" + `"projects/{project}/subscriptions/{subscription}"` + "`" + `. ` + "`" + `{subscription}` + "`" + ` must start with a letter, and contain only letters (` + "`" + `[A-Za-z]` + "`" + `), numbers (` + "`" + `[0-9]` + "`" + `), dashes (` + "`" + `-` + "`" + `), underscores (` + "`" + `_` + "`" + `), periods (` + "`" + `.` + "`" + `), tildes (` + "`" + `~` + "`" + `), plus (` + "`" + `+` + "`" + `) or percent signs (` + "`" + `%` + "`" + `). It must be between 3 and 255 characters in length, and it must not start with ` + "`" + `"goog"` + "`" + `.
      },
    ],
  }</pre>
</div>

<div class="method">
    <code class="details" id="list_next">list_next(previous_request, previous_response)</code>
  <pre>Retrieves the next page of results.

Args:
  previous_request: The request for the previous page. (required)
  previous_response: The response from the request for the previous page. (required)

Returns:
  A request object that you can call 'execute()' on to request the next
  page. Returns None if there are no more items in the collection.
    </pre>
</div>

</body></html>`,
		},
	}
	want := langutil.MethodParamSets{
		{APIName: "pubsub", APIVersion: "v1", FragmentName: "projects.subscriptions.create"}: {
			{Name: "name", Type: "string"},
			{Name: "list_ex", Type: "list"},
			{Name: "body", Type: "object"},
		},
		{APIName: "pubsub", APIVersion: "v1", FragmentName: "projects.subscriptions.list"}: {
			{Name: "project_id", Type: "string"},
		},
	}
	for filename := range ctx.fs.(filesys.MapFS) {
		filenamePattern := regexp.MustCompile(`.*/(\w+)_(v\w+)\.[^/]*`)
		apiNameVersion := filenamePattern.FindStringSubmatch(filename)
		if apiNameVersion == nil {
			t.Fatal(fmt.Errorf("filename of unexpected pattern: %v", filename))
		}
		apiName := string(apiNameVersion[1])
		apiVersion := string(apiNameVersion[2])
		if err := parseFile(&ctx, apiName, apiVersion, filename); err != nil {
			t.Fatal(err)
		}
	}
	if got := ctx.MethodParamSets; !reflect.DeepEqual(got, want) {
		t.Errorf("wrong MethodParamSets, got:\n%v\nwant:\n%v", got, want)
	}
}
