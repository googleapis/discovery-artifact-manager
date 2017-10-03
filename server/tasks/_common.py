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

"""Contains common functions."""


def check_prerelease(repo, latest_tag, github_account, force):
    """Returns true if the repo is in a valid state for release.

    Checks that:
        1. there has been at least one commit since the last tag.
        2. all authors since the last tag were `github_account`, unless `force`
            is true.

    Args:
        repo (Repository): the repo to validate.
        github_account (GitHubAccount, optional): the GitHub account to
            validate author emails against.
        latest_tag (str): the latest tag.
        force (bool, optional): if true, the check that all authors since the
            last tag were `github_account` is ignored.

    Returns:
        bool: whether or not to proceed with a release.
    """
    authors = repo.authors_since(latest_tag)
    if not authors:
        return False
    if force:
        return True
    if not all([x == github_account.email for x in authors]):
        return False
    return True
