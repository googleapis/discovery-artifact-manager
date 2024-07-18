# Introduction

The Discovery Artifact Manager is intended to facilitate testing, publishing, and synchronization of generators
and artifacts for client libraries and generated code samples of Google APIs defined by the API Discovery Service.

## Discovery doc cache

To aid hermetic testing of client libraries and samples (avoiding synchronization issues), the `discoveries`
directory hosts a local cache of Discovery docs from the [API Discovery Service](https://developers.google.com/discovery/).

This cache is updated by an internal mechanism and cannot be run locally at this time. Discovery files are expected
(but not guaranteed) to be updated O(day) from availability in the API Discovery Service. These documents are only
updated if they materially change and are normalized (sorted keys) to make reviewing diffs possible.

## Discovery based clients

Discovery-based client library code is not available in this repository.

* .NET - https://github.com/googleapis/google-api-dotnet-client
* Go - https://github.com/googleapis/google-api-go-client
* Java - https://github.com/googleapis/google-api-java-client-services
* Javascript/Typescript - https://github.com/googleapis/google-api-nodejs-client
* Ruby - https://github.com/googleapis/google-api-ruby-client
* PHP - https://github.com/googleapis/google-api-php-client-services
* Python - https://github.com/googleapis/google-api-python-client

## Deprecations

- [DEPRECATED] the copy of the Google API client library generator code in the `google-api-client-generator` folder
- [DEPRECATED] the copy of the code samples generator [toolkit](https://github.com/googleapis/toolkit/) in the `toolkit` folder
