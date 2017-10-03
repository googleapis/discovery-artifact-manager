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
from unittest.mock import Mock, call, mock_open, patch

from tasks import discovery_artifact_manager
from tests import common

_DISCOVERY_FILENAMES = [
    'discoveries/discovery.v1.json',
    'discoveries/admin.directory_v1.json',
    'discoveries/admin.datatransfer_v1.json',
    'discoveries/foo.v1.json',
    'discoveries/bar.v1.json',
    'discoveries/baz.v1.json',
    'discoveries/index.json'
]


@patch('tasks.discovery_artifact_manager.open', new_callable=mock_open)
@patch('tasks.discovery_artifact_manager.glob')
@patch('tasks.discovery_artifact_manager._git.clone_from_github')
def test_discovery_documents(clone_from_github_mock, glob_mock, open_mock):
    repo_mock = Mock()
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect
    glob_mock.glob.return_value = _DISCOVERY_FILENAMES
    open_mock.side_effect = [
        mock_open(read_data='{"id":"discovery:v1"}').return_value,
        mock_open(read_data='{"id":"admin:directory_v1"}').return_value,
        mock_open(read_data='{"id":"admin:datatransfer_v1"}').return_value,
        mock_open(read_data='{"id":"foo:v1"}').return_value,
        mock_open(read_data='{"id":"foo:v1"}').return_value,
        mock_open(read_data='{"id":"baz:v1"}').return_value
    ]
    assert discovery_artifact_manager.discovery_documents('/tmp') == {
        'discovery:v1': 'discoveries/discovery.v1.json',
        'admin:directory_v1': 'discoveries/admin.directory_v1.json',
        'admin:datatransfer_v1': 'discoveries/admin.datatransfer_v1.json',
        'foo:v1': 'discoveries/foo.v1.json',
        'baz:v1': 'discoveries/baz.v1.json'
    }


@patch('tasks.discovery_artifact_manager.open', new_callable=mock_open)
@patch('tasks.discovery_artifact_manager.glob')
@patch('tasks.discovery_artifact_manager._git.clone_from_github')
def test_discovery_documents_preferred(clone_from_github_mock,
                                       glob_mock,
                                       open_mock):
    repo_mock = Mock()
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect
    glob_mock.glob.return_value = _DISCOVERY_FILENAMES
    open_mock.side_effect = [
        mock_open(read_data='{"id":"discovery:v1"}').return_value,
        mock_open(read_data='{"id":"admin:directory_v1"}').return_value,
        mock_open(read_data='{"id":"admin:datatransfer_v1"}').return_value,
        mock_open(read_data='{"id":"foo:v1"}').return_value,
        # Test that a Discovery document with the same ID as a previous one
        # isn't returned.
        mock_open(read_data='{"id":"foo:v1"}').return_value,
        mock_open(read_data='{"id":"baz:v1"}').return_value,
        mock_open(read_data=(
            '{"items":['
            '{"id":"discovery:v1","preferred":true},'
            '{"id":"admin:directory_v1","preferred":false},'
            '{"id":"admin:datatransfer_v1","preferred":false},'
            '{"id":"foo:v1","preferred":true},'
            '{"id":"baz:v1","preferred":false}]}')).return_value
    ]
    assert discovery_artifact_manager.discovery_documents(
        '/tmp', preferred=True) == {
            'discovery:v1': 'discoveries/discovery.v1.json',
            'admin:directory_v1': 'discoveries/admin.directory_v1.json',
            'admin:datatransfer_v1': 'discoveries/admin.datatransfer_v1.json',
            'foo:v1': 'discoveries/foo.v1.json'
        }


@patch('tasks.discovery_artifact_manager.open', new_callable=mock_open)
@patch('tasks.discovery_artifact_manager.glob')
@patch('tasks.discovery_artifact_manager._git.clone_from_github')
def test_discovery_documents_skip(clone_from_github_mock,
                                  glob_mock,
                                  open_mock):
    repo_mock = Mock()
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect
    glob_mock.glob.return_value = _DISCOVERY_FILENAMES
    open_mock.side_effect = [
        mock_open(read_data='{"id":"discovery:v1"}').return_value,
        mock_open(read_data='{"id":"admin:directory_v1"}').return_value,
        mock_open(read_data='{"id":"admin:datatransfer_v1"}').return_value,
        mock_open(read_data='{"id":"foo:v1"}').return_value,
        mock_open(read_data='{"id":"foo:v1"}').return_value,
        mock_open(read_data='{"id":"baz:v1"}').return_value
    ]
    assert discovery_artifact_manager.discovery_documents(
        '/tmp', skip=['discovery:v1']) == {
            'admin:directory_v1': 'discoveries/admin.directory_v1.json',
            'admin:datatransfer_v1': 'discoveries/admin.datatransfer_v1.json',
            'foo:v1': 'discoveries/foo.v1.json',
            'baz:v1': 'discoveries/baz.v1.json'
        }


@patch('tasks.discovery_artifact_manager.open', new_callable=mock_open)
@patch('tasks.discovery_artifact_manager.glob')
@patch('tasks.discovery_artifact_manager._git.clone_from_github')
def test_discovery_documents_preferred_skip(clone_from_github_mock,
                                            glob_mock,
                                            open_mock):
    repo_mock = Mock()
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect
    glob_mock.glob.return_value = _DISCOVERY_FILENAMES
    open_mock.side_effect = [
        mock_open(read_data='{"id":"discovery:v1"}').return_value,
        mock_open(read_data='{"id":"admin:directory_v1"}').return_value,
        mock_open(read_data='{"id":"admin:datatransfer_v1"}').return_value,
        mock_open(read_data='{"id":"foo:v1"}').return_value,
        mock_open(read_data='{"id":"foo:v1"}').return_value,
        mock_open(read_data='{"id":"baz:v1"}').return_value,
        mock_open(read_data=(
            '{"items":['
            '{"id":"discovery:v1","preferred":true},'
            '{"id":"admin:directory_v1","preferred":false},'
            '{"id":"admin:datatransfer_v1","preferred":false},'
            '{"id":"foo:v1","preferred":true},'
            '{"id":"baz:v1","preferred":false}]}')).return_value
    ]
    assert discovery_artifact_manager.discovery_documents(
        '/tmp', preferred=True, skip=['discovery:v1']) == {
            'admin:directory_v1': 'discoveries/admin.directory_v1.json',
            'admin:datatransfer_v1': 'discoveries/admin.datatransfer_v1.json',
            'foo:v1': 'discoveries/foo.v1.json',
        }


@patch('tasks.discovery_artifact_manager.os.environ')
@patch('tasks.discovery_artifact_manager.check_output')
@patch('tasks.discovery_artifact_manager.os.makedirs')
@patch('tasks.discovery_artifact_manager.TemporaryDirectory')
@patch('tasks.discovery_artifact_manager._git.clone_from_github')
def test_update(clone_from_github_mock,
                temporary_directory_mock,
                makedirs_mock,
                check_output_mock,
                environ_mock):
    repo_mock = Mock()
    repo_mock.diff_name_status.return_value = [
        'discoveries/index.json',
        'discoveries/foo.v1.json'
    ]
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect
    temporary_directory_mock.return_value.__enter__.return_value = '/tmp/go'
    environ_mock.copy.return_value = {}

    manager = Mock()
    manager.attach_mock(clone_from_github_mock, 'clone_from_github')
    manager.attach_mock(makedirs_mock, 'makedirs')
    manager.attach_mock(check_output_mock, 'check_output')
    manager.attach_mock(repo_mock, 'repo')

    discovery_artifact_manager.update('/tmp', common.GITHUB_ACCOUNT)
    assert manager.mock_calls == [
        call.clone_from_github('googleapis/discovery-artifact-manager',
                               '/tmp/discovery-artifact-manager',
                               github_account=common.GITHUB_ACCOUNT),
        call.makedirs('/tmp/go/src'),
        call.check_output(['ln', '-s',
                           '/tmp/discovery-artifact-manager/src',
                           '/tmp/go/src/discovery-artifact-manager']),
        call.check_output(['go', 'run', 'src/main/updatedisco/main.go'],
                          cwd='/tmp/discovery-artifact-manager',
                          env={'GOPATH': '/tmp/go'}),
        call.repo.add(['discoveries']),
        call.repo.diff_name_status(),
        call.repo.commit('Autogenerated Discovery document update',
                         'Alice',
                         'alice@test.com'),
        call.repo.push()
    ]


@patch('tasks.discovery_artifact_manager.os.environ')
@patch('tasks.discovery_artifact_manager.check_output')
@patch('tasks.discovery_artifact_manager.os.makedirs')
@patch('tasks.discovery_artifact_manager.TemporaryDirectory')
@patch('tasks.discovery_artifact_manager._git.clone_from_github')
def test_update_no_changes(clone_from_github_mock,
                           temporary_directory_mock,
                           makedirs_mock,
                           check_output_mock,
                           environ_mock):
    repo_mock = Mock()
    repo_mock.diff_name_status.return_value = []
    side_effect = common.clone_from_github_mock_side_effect(repo_mock)
    clone_from_github_mock.side_effect = side_effect
    temporary_directory_mock.return_value.__enter__.return_value = '/tmp/go'
    environ_mock.copy.return_value = {}

    manager = Mock()
    manager.attach_mock(clone_from_github_mock, 'clone_from_github')
    manager.attach_mock(makedirs_mock, 'makedirs')
    manager.attach_mock(check_output_mock, 'check_output')
    manager.attach_mock(repo_mock, 'repo')

    discovery_artifact_manager.update('/tmp', common.GITHUB_ACCOUNT)
    assert manager.mock_calls == [
        call.clone_from_github('googleapis/discovery-artifact-manager',
                               '/tmp/discovery-artifact-manager',
                               github_account=common.GITHUB_ACCOUNT),
        call.makedirs('/tmp/go/src'),
        call.check_output(['ln', '-s',
                           '/tmp/discovery-artifact-manager/src',
                           '/tmp/go/src/discovery-artifact-manager']),
        call.check_output(['go', 'run', 'src/main/updatedisco/main.go'],
                          cwd='/tmp/discovery-artifact-manager',
                          env={'GOPATH': '/tmp/go'}),
        call.repo.add(['discoveries']),
        call.repo.diff_name_status()
    ]
