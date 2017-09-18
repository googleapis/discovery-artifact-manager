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
"""Contains common functions and constants for tests."""
from tasks import accounts

GITHUB_ACCOUNT = accounts.GitHubAccount('Test', 'test@test.com', '_', '_')


def clone_from_github_mock_side_effect(repo_mock):
    """Returns a side effect function for clone_from_github.

    The function returned returns repo_mock with its filepath set to the
    filepath that clone_from_github was called with.

    Args:
        repo_mock (Mock): the repo to be returned from clone_from_github.

    Returns:
        function(str, str, github_account=GitHubAccount): a side effect
            function for clone_from_github.
    """
    def side_effect(path, dest, github_account=None):
        repo_mock.filepath = dest
        return repo_mock
    return side_effect
