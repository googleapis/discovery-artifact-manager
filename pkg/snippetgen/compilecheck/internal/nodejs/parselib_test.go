package nodejs

import (
	"reflect"
	"testing"

	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/compilecheck/internal/filesys"
	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/compilecheck/internal/langutil"
)

func TestParseLibs(t *testing.T) {
	tests := []struct {
		fname, content string
		params         langutil.MethodParamSets
	}{
		{
			fname: "/lib/myservice/v2.ts",
			params: langutil.MethodParamSets{
				{"myservice", "v2", "myService.myMethod"}: {
					{"foo", "string"},
					{"bar", "object"},
					{"bar.zip", "string"},
					{"bar.zap", "number"},
				},
			},
			content: `
		/**
		 * myService.myMethod
		 * @param {object} params Parameters for request
		 * @param {string} params.foo
		 * @param {object} params.bar
		 * @param {string} params.bar.zip
		 * @param {number} params.bar.zap
		 */
		 `,
		},
		{
			fname: "/lib/myservice/v1.ts",
			params: langutil.MethodParamSets{
				{"myservice", "v1", "appengine.apps.get"}: {
					{"appsId", "string"},
				},
			},
			content: `
/**
 * Google App Engine Admin API
 *
 * Provisions and manages App Engine applications.
 *
 * @example
 * var google = require('googleapis');
 * var appengine = google.appengine('v1beta4');
 *
 * @namespace appengine
 * @type {Function}
 * @version v1beta4
 * @variation v1beta4
 * @param {object=} options Options for Appengine
 */
function Appengine(options) { // eslint-disable-line
  var self = this;
  self._options = options || {};

  self.apps = {

    /**
     * appengine.apps.get
     *
     * @desc Gets information about an application.
     *
     * @alias appengine.apps.get
     * @memberOf! appengine(v1beta4)
     *
     * @param {object} params Parameters for request
     * @param {string} params.appsId Part of name. Name of the application to get. For example: "apps/myapp".
     * @param {boolean=} params.ensureResourcesExist Certain resources associated with an application are created on-demand.
     * @param {callback} callback The callback that handles the response.
     * @return {object} Request object
     */
    get: function (params, callback) {`,
		},
	}

	for _, tst := range tests {
		opener := filesys.MapFS{
			tst.fname: tst.content,
		}
		sampleMethods := map[langutil.MethodID]string{}
		for mid := range tst.params {
			sampleMethods[mid] = ""
		}
		result, err := parseLibs(sampleMethods, "/lib", opener)
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(result, tst.params) {
			t.Errorf("%v: got %v, want %v", tst.fname, result, tst.params)
		}
	}
}
