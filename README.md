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

# Modifying a client library repo

To make changes to a repo, use the `git subrepo pull` and `git subrepo push` commands. The former will merge your local client with fetched upstream changes, and the allter will actually do the push to the upstream sub-repo. For example, to push the PHP client library:

``` shell
git subrepo pull clients/php/google-api-php-client-services
git subrepo push clients/php/google-api-php-client-services
```

If you need to reset your HEAD locally after a subrepo push, then you would not want to subsequently pull again (since that will merge the upstream changes you pushed earlier). Instead, you can force-push your chnages by using `git subrepo push --force`.




