package common

import (
	"fmt"
	"testing"

	"discovery-artifact-manager/main/common"
)

func replaceFail(t *testing.T, original, want, got string) {
	t.Errorf(`Replacement failed:
Old:
%s
New (want):
%s
New (got):
%s`, original, want, got)
}

const (
	index = `<img src="https://avatars0.githubusercontent.com/u/1342004?v=3&s=96" alt="Google Inc. logo" title="Google" align="right" height="96" width="96"/>

### google-api-nodejs-client API Reference Docs

* [v19.0.0 (latest)](http://google.github.io/google-api-nodejs-client/19.0.0/index.html)
* [v18.1.0](http://google.github.io/google-api-nodejs-client/18.1.0/index.html)
* [v18.0.0](http://google.github.io/google-api-nodejs-client/18.0.0/index.html)
`
	indexWant = `<img src="https://avatars0.githubusercontent.com/u/1342004?v=3&s=96" alt="Google Inc. logo" title="Google" align="right" height="96" width="96"/>

### google-api-nodejs-client API Reference Docs

* [v19.1.0 (latest)](http://google.github.io/google-api-nodejs-client/19.1.0/index.html)
* [v19.0.0](http://google.github.io/google-api-nodejs-client/19.0.0/index.html)
* [v18.1.0](http://google.github.io/google-api-nodejs-client/18.1.0/index.html)
* [v18.0.0](http://google.github.io/google-api-nodejs-client/18.0.0/index.html)
`
	docLinkFormat = `* [v%v.%v.%v%s](http://google.github.io/google-api-nodejs-client/%s/index.html)
`
	version = "19.1.0"
)

func TestReplacePattern(t *testing.T) {
	num, _ := common.Version(version)
	indexGot, _, _ := common.ReplacePattern([]byte(index), docLinkFormat,
		fmt.Sprintf(docLinkFormat, num[1], num[2], num[3], "$4", version)+
			fmt.Sprintf(docLinkFormat, "$1", "$2", "$3", "", "$5"))
	if string(indexGot) != indexWant {
		replaceFail(t, index, indexWant, string(indexGot))
	}
}

const (
	contents = `{
  "name": "googleapis",
  "version": "19.0.0",
  "author": "Google Inc.",
  "license": "Apache-2.0",
  "description": "Google APIs Client Library for Node.js",
  "engines": {
    "node": ">=0.10"
  }
}`
	contentsWant = `{
  "name": "googleapis",
  "version": "19.1.0",
  "author": "Google Inc.",
  "license": "Apache-2.0",
  "description": "Google APIs Client Library for Node.js",
  "engines": {
    "node": ">=0.10"
  }
}`
)

func TestReplaceValue(t *testing.T) {
	contentsGot, _ := common.ReplaceValue([]byte(contents), "version", version)
	if string(contentsGot) != contentsWant {
		replaceFail(t, contents, contentsWant, string(contentsGot))
	}
}
