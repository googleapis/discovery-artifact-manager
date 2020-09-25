# Copyright 2017, Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Contains definitions/getters for GitHub and package manager accounts."""

import os
from collections import namedtuple

from google.cloud import datastore

GitHubAccount = namedtuple('GitHubAccount',
                           'name email username personal_access_token')
NpmAccount = namedtuple('NpmAccount', 'auth_token')
RubyGemsAccount = namedtuple('RubyGemsAccount', 'api_key')


def _get(type_):
    client = datastore.Client()
    obj = list(client.query(kind=type_.__name__).fetch())[0]
    return type_(*[obj[x] for x in type_._fields])


def get_github_account():
    """Returns the GitHub account stored in Datastore.

    Returns:
        GitHubAccount: a GitHub account.
    """
    # Allow environment variables to set github details for
    # easy local debugging.
    env = os.environ  # for brevity
    github_token = env.get("GITHUB_TOKEN")
    if github_token:
        return GitHubAccount(
            env["GITHUB_USER"], 
            env["GITHUB_EMAIL"],
            env["GITHUB_USERNAME"],
            github_token)
    return _get(GitHubAccount)


def get_npm_account():
    """Returns the npm account stored in Datastore.

    Returns:
        NpmAccount: an npm account.
    """
    return _get(NpmAccount)


def get_rubygems_account():
    """Returns the RubyGems account stored in Datastore.

    Returns:
        RubyGemsAccount: a RubyGems account.
    """
    return _get(RubyGemsAccount)
