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
from collections import namedtuple

from google.cloud import datastore

GitHubAccount = namedtuple('GitHubAccount',
                           'name email username personal_access_token')
NpmAccount = namedtuple('NpmAccount', 'auth_token')
RubyGemsAccount = namedtuple('RubyGemsAccount', 'api_key')


def _obj(kind):
    client = datastore.Client()
    return list(client.query(kind=kind).fetch())[0]


def get_github_account():
    """Returns the GitHub account stored in Datastore.

    Returns:
        GitHubAccount: a GitHub account.
    """
    obj = _obj('GitHubAccount')
    return GitHubAccount(obj['name'], obj['email'], obj['username'],
                         obj['personal_access_token'])


def get_npm_account():
    """Returns the npm account stored in Datastore.

    Returns:
        NpmAccount: an npm account.
    """
    obj = _obj('NpmAccount')
    return NpmAccount(obj['auth_token'])


def get_rubygems_account():
    """Returns the RubyGems account stored in Datastore.

    Returns:
        RubyGemsAccount: a RubyGems account.
    """
    obj = _obj('RubyGemsAccount')
    return RubyGemsAccount(obj['api_key'])
