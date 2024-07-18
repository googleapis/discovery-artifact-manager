# Introduction

The Discovery Artifact Manager is intended to facilitate testing, publishing, and synchronization of generators
and artifacts for client libraries and generated code samples of Google APIs defined by the API Discovery Service.

## Discovery doc cache

To aid hermetic testing of client libraries and samples (avoiding synchronization issues), the `discoveries`
directory hosts a local cache of Discovery docs from the [API Discovery Service](https://developers.google.com/discovery/).

This cache is updated by an internal mechanism and cannot be run locally at this time. Discovery files are expected
(but not guaranteed) to be updated O(day) from availability in the API Discovery Service. These documents are only
updated if they materially change and are normalized (sorted keys) to make reviewing diffs possible.

## Deprecations

- [DEPRECATED] the Google API client library generator (used to generate Java and PHP client libraries)
- [DEPRECATED] some of the [Discovery-based Google API client libraries](https://developers.google.com/discovery/libraries), along with their generators
- [DEPRECATED] the code generation [toolkit](https://github.com/googleapis/toolkit/) used to generate code samples for client libraries.
