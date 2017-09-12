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
from unittest.mock import Mock, patch

from tasks import accounts


@patch('tasks.accounts.datastore.Client', autospec=True)
def test_get_github_account(client_mock):
    query_mock = Mock()
    query_mock.return_value.fetch.return_value = [{
        'name': 'Test',
        'email': 'test@example.com',
        'username': 'test',
        'personal_access_token': 'token'
    }]
    client_mock.return_value.query = query_mock
    expected = accounts.GitHubAccount(
        'Test', 'test@example.com', 'test', 'token')
    actual = accounts.get_github_account()
    query_mock.assert_called_once_with(kind='GitHubAccount')
    assert actual == expected


@patch('tasks.accounts.datastore.Client', autospec=True)
def test_get_npm_account(client_mock):
    query_mock = Mock()
    query_mock.return_value.fetch.return_value = [{'auth_token': 'token'}]
    client_mock.return_value.query = query_mock
    expected = accounts.NpmAccount('token')
    actual = accounts.get_npm_account()
    query_mock.assert_called_once_with(kind='NpmAccount')
    assert actual == expected


@patch('tasks.accounts.datastore.Client', autospec=True)
def test_get_rubygems_account(client_mock):
    query_mock = Mock()
    query_mock.return_value.fetch.return_value = [{'api_key': 'key'}]
    client_mock.return_value.query = query_mock
    expected = accounts.RubyGemsAccount('key')
    actual = accounts.get_rubygems_account()
    query_mock.assert_called_once_with(kind='RubyGemsAccount')
    assert actual == expected
