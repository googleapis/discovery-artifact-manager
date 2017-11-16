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
from unittest.mock import patch

from tasks import _git, accounts

_REPO = _git.Repository('/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_clone_from_github(check_output_mock):
    repo = _git.clone_from_github('example/myrepo', '/tmp')
    check_output_mock.assert_called_once_with(
        ['git', 'clone', 'https://github.com/example/myrepo', '/tmp'])
    assert repo.filepath == _git.Repository('/tmp').filepath

    check_output_mock.reset_mock()
    github_account = accounts.GitHubAccount(
        'Alice', 'alice@test.com', 'test', 'token')
    repo = _git.clone_from_github(
        'example/myrepo', '/tmp', github_account=github_account)
    check_output_mock.assert_called_once_with([
        'git', 'clone', 'https://test:token@github.com/example/myrepo', '/tmp'
    ])
    assert repo.filepath == _git.Repository('/tmp').filepath


@patch('tasks._git.check_output', autospec=True)
def test_repository_add(check_output_mock):
    _REPO.add(['src'])
    check_output_mock.assert_called_once_with(
        ['git', 'add', 'src'], cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_add_multiple_filepaths(check_output_mock):
    _REPO.add(['hello', 'world'])
    check_output_mock.assert_called_once_with(
        ['git', 'add', 'hello', 'world'], cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_authors_since_none(check_output_mock):
    check_output_mock.return_value = ''
    authors = _REPO.authors_since('HEAD~1')
    check_output_mock.assert_called_once_with(
        ['git', 'log', 'HEAD~1..HEAD', '--pretty=format:%ae'], cwd='/tmp')
    assert authors == []


@patch('tasks._git.check_output', autospec=True)
def test_repository_authors_since_one(check_output_mock):
    check_output_mock.return_value = 'alice@test.com'
    authors = _REPO.authors_since('HEAD~1')
    check_output_mock.assert_called_once_with(
        ['git', 'log', 'HEAD~1..HEAD', '--pretty=format:%ae'], cwd='/tmp')
    assert authors == ['alice@test.com']


@patch('tasks._git.check_output', autospec=True)
def test_repository_authors_since_multiple(check_output_mock):
    check_output_mock.return_value = 'alice@test.com\nbob@test.com'
    authors = _REPO.authors_since('HEAD~2')
    check_output_mock.assert_called_once_with(
        ['git', 'log', 'HEAD~2..HEAD', '--pretty=format:%ae'], cwd='/tmp')
    assert authors == ['alice@test.com', 'bob@test.com']


@patch('tasks._git.check_output', autospec=True)
def test_repository_checkout(check_output_mock):
    _REPO.checkout('docs')
    check_output_mock.assert_called_once_with(
        ['git', 'checkout', 'docs'], cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_commit(check_output_mock):
    _REPO.commit('hello world', 'example name', 'example@example.com')
    check_output_mock.assert_called_once_with(
        ['git',
         '-c', 'user.name=example name',
         '-c', 'user.email=example@example.com',
         'commit', '-a', '--allow-empty-message', '-m', 'hello world'],
        cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_diff_name_status_none(check_output_mock):
    check_output_mock.return_value = ''
    pairs = _REPO.diff_name_status()
    check_output_mock.assert_called_once_with(
        ['git', 'diff', '--name-status', '--staged'], cwd='/tmp')
    assert pairs == []


@patch('tasks._git.check_output', autospec=True)
def test_repository_diff_name_status_none_staged(check_output_mock):
    check_output_mock.return_value = ''
    pairs = _REPO.diff_name_status(staged=False)
    check_output_mock.assert_called_once_with(
        ['git', 'diff', '--name-status'], cwd='/tmp')
    assert pairs == []


@patch('tasks._git.check_output', autospec=True)
def test_repository_diff_name_status_none_rev(check_output_mock):
    check_output_mock.return_value = ''
    pairs = _REPO.diff_name_status(rev='HEAD~1')
    check_output_mock.assert_called_once_with(
        ['git', 'diff', '--name-status', 'HEAD~1..HEAD'], cwd='/tmp')
    assert pairs == []


@patch('tasks._git.check_output', autospec=True)
def test_repository_diff_name_status_one(check_output_mock):
    check_output_mock.return_value = 'A\t/tmp/hello.txt'
    pairs = _REPO.diff_name_status()
    assert pairs == [('/tmp/hello.txt', _git.Status.ADDED)]


@patch('tasks._git.check_output', autospec=True)
def test_repository_diff_name_status_multiple(check_output_mock):
    check_output_mock.return_value = ('A\t/tmp/hello.txt\n'
                                      'D\t/tmp/world.txt\n'
                                      'M\t/tmp/foo.txt\n'
                                      'X\t/tmp/bar.txt')
    pairs = _REPO.diff_name_status()
    assert pairs == [('/tmp/hello.txt', _git.Status.ADDED),
                     ('/tmp/world.txt', _git.Status.DELETED),
                     ('/tmp/foo.txt', _git.Status.UPDATED),
                     ('/tmp/bar.txt', _git.Status.UNKNOWN)]


@patch('tasks._git.check_output', autospec=True)
def test_repository_latest_tag(check_output_mock):
    check_output_mock.return_value = 'v0.1'
    tag = _REPO.latest_tag()
    check_output_mock.assert_called_once_with(
        ['git', 'describe', '--tags', '--abbrev=0'], cwd='/tmp')
    assert tag == 'v0.1'


@patch('tasks._git.check_output', autospec=True)
def test_repository_push(check_output_mock):
    _REPO.push()
    check_output_mock.assert_called_once_with(
        ['git', 'push', 'origin', 'master'], cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_push_remote_branch(check_output_mock):
    _REPO.push(remote='github', branch='dev')
    check_output_mock.assert_called_once_with(
        ['git', 'push', 'github', 'dev'], cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_push_tags(check_output_mock):
    _REPO.push(tags=True)
    check_output_mock.assert_called_once_with(
        ['git', 'push', 'origin', '--tags'], cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_push_nokeycheck(check_output_mock):
    _REPO.push(nokeycheck=True)
    check_output_mock.assert_called_once_with(
        ['git', 'push', 'origin', 'master', '--push-option', 'nokeycheck'],
        cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_soft_reset(check_output_mock):
    _REPO.soft_reset('HEAD~1')
    check_output_mock.assert_called_once_with(
        ['git', 'reset', '--soft', 'HEAD~1'], cwd='/tmp')


@patch('tasks._git.check_output', autospec=True)
def test_repository_tag(check_output_mock):
    _REPO.tag('v0.1')
    check_output_mock.assert_called_once_with(
        ['git', 'tag', 'v0.1'], cwd='/tmp')
