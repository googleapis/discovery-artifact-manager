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
from unittest.mock import Mock, call, patch

import pytest

from tasks import _git, google_api_php_client_services
from tests import common


@patch('tasks.google_api_php_client_services._commit_message.date')
@patch('tasks.google_api_php_client_services.check_output')
@patch('tasks.google_api_php_client_services.os.listdir')
@patch('tasks.google_api_php_client_services.TemporaryDirectory')
@patch('tasks.google_api_php_client_services._git.clone_from_github')
def test_update(clone_from_github_mock,
                temporary_directory_mock,
                listdir_mock,
                check_output_mock,
                date_mock):
    repo_mock = Mock()
    repo_mock.diff_name_status.side_effect = [
        [('src/Google/Service/Foo.php', _git.Status.ADDED),
         ('src/Google/Service/Foo/FooBar.php', _git.Status.ADDED)],
        [('src/Google/Service/Bar.php', _git.Status.UPDATED),
         ('src/Google/Service/Bar/BarFoo.php', _git.Status.ADDED)],
        []  # No change to "baz:v1".
    ]
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect
    discovery_documents = {'foo:v1': 'foo.v1.json',
                           'bar:v1': 'bar.v1.json',
                           'baz:v1': 'baz.v1.json'}
    temporary_directory_mock.return_value.__enter__.return_value = '/tmp2'
    listdir_mock.side_effect = [
        ['Foo', 'Foo.php'],
        ['Bar.php', 'Bar'],
        ['Baz.php', 'Baz']
    ]
    date_mock.today.return_value.isoformat.return_value = '2000-01-01'

    manager = Mock()
    manager.attach_mock(clone_from_github_mock, 'clone_from_github')
    manager.attach_mock(check_output_mock, 'check_output')
    manager.attach_mock(repo_mock, 'repo')

    google_api_php_client_services.update(
        '/tmp', discovery_documents, common.GITHUB_ACCOUNT)
    assert manager.mock_calls == [
        call.clone_from_github('google/google-api-php-client-services',
                               '/tmp/google-api-php-client-services',
                               github_account=common.GITHUB_ACCOUNT),
        call.check_output(
            ['virtualenv', '/tmp/google-api-php-client-services/venv']),
        call.check_output(['/tmp/google-api-php-client-services/venv/bin/pip',
                           'install',
                           'google-apis-client-generator==1.4.3']),
        call.check_output(
            ['/tmp/google-api-php-client-services/venv/bin/generate_library',
             '--input=foo.v1.json',
             '--language=php',
             '--language_variant=1.2.0',
             '--output_dir=/tmp2']),
        call.check_output(
            ['rm', '-rf',
             '/tmp/google-api-php-client-services/src/Google/Service/Foo.php',
             '/tmp/google-api-php-client-services/src/Google/Service/Foo']),
        call.check_output(
            ['cp',
             '/tmp2/Foo.php',
             ('/tmp/google-api-php-client-services/src/Google/Service'
              '/Foo.php')]),
        call.check_output(
            ['cp', '-r',
             '/tmp2/Foo',
             '/tmp/google-api-php-client-services/src/Google/Service/Foo']),
        call.repo.add(['src']),
        call.repo.diff_name_status(),
        call.repo.commit('', '_', '_'),
        call.check_output(
            ['/tmp/google-api-php-client-services/venv/bin/generate_library',
             '--input=bar.v1.json',
             '--language=php',
             '--language_variant=1.2.0',
             '--output_dir=/tmp2']),
        call.check_output(
            ['rm', '-rf',
             '/tmp/google-api-php-client-services/src/Google/Service/Bar.php',
             '/tmp/google-api-php-client-services/src/Google/Service/Bar']),
        call.check_output(
            ['cp',
             '/tmp2/Bar.php',
             ('/tmp/google-api-php-client-services/src/Google/Service'
              '/Bar.php')]),
        call.check_output(
            ['cp', '-r',
             '/tmp2/Bar',
             '/tmp/google-api-php-client-services/src/Google/Service/Bar']),
        call.repo.add(['src']),
        call.repo.diff_name_status(),
        call.repo.commit('', '_', '_'),
        call.check_output(
            ['/tmp/google-api-php-client-services/venv/bin/generate_library',
             '--input=baz.v1.json',
             '--language=php',
             '--language_variant=1.2.0',
             '--output_dir=/tmp2']),
        call.check_output(
            ['rm', '-rf',
             '/tmp/google-api-php-client-services/src/Google/Service/Baz.php',
             '/tmp/google-api-php-client-services/src/Google/Service/Baz']),
        call.check_output(
            ['cp',
             '/tmp2/Baz.php',
             ('/tmp/google-api-php-client-services/src/Google/Service'
              '/Baz.php')]),
        call.check_output(
            ['cp', '-r',
             '/tmp2/Baz',
             '/tmp/google-api-php-client-services/src/Google/Service/Baz']),
        call.repo.add(['src']),
        call.repo.diff_name_status(),
        call.check_output(['composer', 'update'],
                          cwd='/tmp/google-api-php-client-services'),
        call.check_output(['vendor/bin/phpunit', '-c', '.'],
                          cwd='/tmp/google-api-php-client-services'),
        call.repo.soft_reset('HEAD~2'),
        call.repo.commit(('Autogenerated update (2000-01-01)\n'
                          '\nAdd:\n- foo:v1\n'
                          '\nUpdate:\n- bar:v1'),
                         'Test',
                         'test@test.com'),
        call.repo.push()
    ]


@patch('tasks.google_api_php_client_services.check_output')
@patch('tasks.google_api_php_client_services.os.listdir')
@patch('tasks.google_api_php_client_services.TemporaryDirectory')
@patch('tasks.google_api_php_client_services._git.clone_from_github')
def test_update_no_changes(clone_from_github_mock,
                           temporary_directory_mock,
                           listdir_mock,
                           check_output_mock):
    repo_mock = Mock()
    repo_mock.diff_name_status.side_effect = [[], [], []]
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect
    discovery_documents = {'foo:v1': 'foo.v1.json'}
    temporary_directory_mock.return_value.__enter__.return_value = '/tmp2'
    listdir_mock.side_effect = [['Foo', 'Foo.php']]

    manager = Mock()
    manager.attach_mock(clone_from_github_mock, 'clone_from_github')
    manager.attach_mock(check_output_mock, 'check_output')
    manager.attach_mock(repo_mock, 'repo')

    google_api_php_client_services.update(
        '/tmp', discovery_documents, common.GITHUB_ACCOUNT)
    assert manager.mock_calls == [
        call.clone_from_github('google/google-api-php-client-services',
                               '/tmp/google-api-php-client-services',
                               github_account=common.GITHUB_ACCOUNT),
        call.check_output(
            ['virtualenv', '/tmp/google-api-php-client-services/venv']),
        call.check_output(['/tmp/google-api-php-client-services/venv/bin/pip',
                           'install',
                           'google-apis-client-generator==1.4.3']),
        call.check_output(
            ['/tmp/google-api-php-client-services/venv/bin/generate_library',
             '--input=foo.v1.json',
             '--language=php',
             '--language_variant=1.2.0',
             '--output_dir=/tmp2']),
        call.check_output(
            ['rm', '-rf',
             '/tmp/google-api-php-client-services/src/Google/Service/Foo.php',
             '/tmp/google-api-php-client-services/src/Google/Service/Foo']),
        call.check_output(
            ['cp',
             '/tmp2/Foo.php',
             ('/tmp/google-api-php-client-services/src/Google/Service'
              '/Foo.php')]),
        call.check_output(
            ['cp', '-r',
             '/tmp2/Foo',
             '/tmp/google-api-php-client-services/src/Google/Service/Foo']),
        call.repo.add(['src']),
        call.repo.diff_name_status()
    ]


@patch('tasks.google_api_php_client_services.check_output')
@patch('tasks.google_api_php_client_services._git.clone_from_github')
def test_release(clone_from_github_mock, check_output_mock):
    repo_mock = Mock()
    repo_mock.latest_tag.return_value = 'v0.1'
    repo_mock.authors_since.return_value = ['test@test.com', 'test@test.com']
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect

    manager = Mock()
    manager.attach_mock(clone_from_github_mock, 'clone_from_github')
    manager.attach_mock(check_output_mock, 'check_output')
    manager.attach_mock(repo_mock, 'repo')

    google_api_php_client_services.release('/tmp', common.GITHUB_ACCOUNT)
    assert manager.mock_calls == [
        call.clone_from_github('google/google-api-php-client-services',
                               '/tmp/google-api-php-client-services',
                               github_account=common.GITHUB_ACCOUNT),
        call.repo.latest_tag(),
        call.repo.authors_since('v0.1'),
        call.check_output(['composer', 'update'],
                          cwd='/tmp/google-api-php-client-services'),
        call.check_output(['vendor/bin/phpunit', '-c', '.'],
                          cwd='/tmp/google-api-php-client-services'),
        call.repo.tag('v0.2'),
        call.repo.push(tags=True)
    ]


@patch('tasks.google_api_php_client_services._git.clone_from_github')
def test_release_no_commits_since_latest_tag(clone_from_github_mock):
    repo_mock = Mock()
    repo_mock.latest_tag.return_value = 'v0.1'
    repo_mock.authors_since.return_value = []
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect

    manager = Mock()
    manager.attach_mock(clone_from_github_mock, 'clone_from_github')
    manager.attach_mock(repo_mock, 'repo')

    google_api_php_client_services.release('/tmp', common.GITHUB_ACCOUNT)
    assert manager.mock_calls == [
        call.clone_from_github('google/google-api-php-client-services',
                               '/tmp/google-api-php-client-services',
                               github_account=common.GITHUB_ACCOUNT),
        call.repo.latest_tag(),
        call.repo.authors_since('v0.1')
    ]


@patch('tasks.google_api_php_client_services._git.clone_from_github')
def test_release_different_authors_since_latest_tag(clone_from_github_mock):
    repo_mock = Mock()
    repo_mock.latest_tag.return_value = 'v0.1'
    repo_mock.authors_since.return_value = ['test@test.com', 'test2@test.com']
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect

    manager = Mock()
    manager.attach_mock(clone_from_github_mock, 'clone_from_github')
    manager.attach_mock(repo_mock, 'repo')

    google_api_php_client_services.release('/tmp', common.GITHUB_ACCOUNT)
    assert manager.mock_calls == [
        call.clone_from_github('google/google-api-php-client-services',
                               '/tmp/google-api-php-client-services',
                               github_account=common.GITHUB_ACCOUNT),
        call.repo.latest_tag(),
        call.repo.authors_since('v0.1')
    ]


@patch('tasks.google_api_php_client_services._git.clone_from_github')
def test_release_invalid_latest_tag(clone_from_github_mock):
    repo_mock = Mock()
    repo_mock.latest_tag.return_value = 'v1.0'
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect

    manager = Mock()
    manager.attach_mock(clone_from_github_mock, 'clone_from_github')
    manager.attach_mock(repo_mock, 'repo')

    with pytest.raises(Exception) as excinfo:
        google_api_php_client_services.release('/tmp', common.GITHUB_ACCOUNT)
    assert str(excinfo.value) == ('latest tag does not match the pattern'
                                  r' "^v0\.([0-9]+)$": v1.0')
    assert manager.mock_calls == [
        call.clone_from_github('google/google-api-php-client-services',
                               '/tmp/google-api-php-client-services',
                               github_account=common.GITHUB_ACCOUNT),
        call.repo.latest_tag()
    ]
