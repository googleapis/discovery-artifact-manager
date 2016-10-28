# Introduction

The Discovery Artifact Manager is intended to facilitate testing, publishing,
and synchronization of Toolkit, discovery docs from API explorer, and discovery
based Google client libraries.

# Local machine setup

Install [git-subrepo](https://github.com/ingydotnet/git-subrepo) on your local machine.


# Adding a new client library repo

Use the `git subrepo clone` command, from the root directory of this repository. The NodeJS library, for example, is installed using:

``` shell
git subrepo clone https://github.com/google/google-api-nodejs-client.git clients/nodejs/google-api-nodejs-client
```



