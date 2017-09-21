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

"""Contains update/release functions for google-api-nodejs-client."""

import json
import os
import re
from datetime import date
from os.path import join

from tasks import _commit_message, _git
from tasks._check_output import check_output

_REPO_NAME = 'google-api-nodejs-client'
_REPO_PATH = 'google/google-api-nodejs-client'
# Matches strings like "apis/foo/v1.ts".
_SERVICE_FILENAME_RE = re.compile(r'apis/(.+)/(.+)\.ts')
# Matches strings like "20.3.1".
_VERSION_RE = re.compile(r'^([0-9]+)\.([0-9]+)\.([0-9]+)$')


class _Version(object):
    """Represents a version of the format "1.2.3"."""

    def __init__(self, latest_tag):
        match = _VERSION_RE.match(latest_tag)
        if not match:
            raise Exception(
                'latest tag does not match the pattern "{}": {}'.format(
                    _VERSION_RE.pattern, latest_tag))
        self.major_version = int(match.group(1))
        self.minor_version = int(match.group(2))

    def __str__(self):
        return '{}.{}.0'.format(
            self.major_version, self.minor_version)

    def bump_major(self):
        self.major_version += 1
        self.minor_version = 0

    def bump_minor(self):
        self.minor_version += 1


def _build(repo):
    check_output(['node', '--max_old_space_size=2000', '/usr/bin/npm', 'run',
                  'build'],
                 cwd=repo.filepath)


def _generate_all_clients(repo):
    check_output(['node', '--max_old_space_size=2000', '/usr/bin/npm', 'run',
                  'generate-apis'],
                 cwd=repo.filepath)
    added, deleted, updated = set(), set(), set()
    status_to_ids = {
        _git.Status.ADDED: added,
        _git.Status.DELETED: deleted,
        _git.Status.UPDATED: updated
    }
    for filename, status in repo.diff_name_status(staged=False):
        match = _SERVICE_FILENAME_RE.match(filename)
        if not match:
            continue
        name_version = '{}:{}'.format(match.group(1), match.group(2))
        status_to_ids.get(status, set()).add(name_version)
    return added, deleted, updated


def _check_latest_version(latest_tag):
    latest_version = check_output(['npm', 'view', 'googleapis', 'version'])
    if latest_tag != latest_version:
        raise Exception(
            ('latest tag does not match the latest package version on npm:'
             ' {} != {}').format(latest_tag, latest_version))


def _install_dependencies(repo):
    check_output(['npm', 'install'], cwd=repo.filepath)


def _publish_package(repo, npm_account):
    with open(os.path.expanduser('~/.npmrc'), 'w') as file_:
        file_.write('//registry.npmjs.org/:_authToken={}\n'.format(
            npm_account.auth_token))
    check_output(['npm', 'publish'], cwd=repo.filepath)


def _run_tests(repo):
    check_output(['npm', 'run', 'test'], cwd=repo.filepath)


def _update_changelog_md(repo, new_version, added, deleted, updated):
    data = '##### {} - {}\n'.format(
        new_version, date.today().strftime("%d %B %Y"))
    if deleted:
        data += '\n###### Breaking changes\n'
    for name_version in sorted(deleted):
        data += '- Deleted `{}`\n'.format(name_version)
    if added or updated:
        data += '\n###### Backwards compatible changes\n'
    for name_version in sorted(added):
        data += '- Added `{}`\n'.format(name_version)
    for name_version in sorted(updated):
        data += '- Updated `{}`\n'.format(name_version)
    data += '\n'
    filename = join(repo.filepath, 'CHANGELOG.md')
    with open(filename) as file_:
        data = data + file_.read()
    with open(filename, 'w') as file_:
        file_.write(data)


def _update_and_publish_gh_pages(repo, new_version, github_account):
    check_output(['npm', 'run', 'doc'], cwd=repo.filepath)
    repo.checkout('gh-pages')
    check_output(['rm', '-rf', 'latest'], cwd=repo.filepath)
    doc_filepath = 'doc/googleapis/{}'.format(new_version)
    check_output(['cp', '-r', 'doc/googleapis/{}'.format(new_version)],
                 cwd=repo.filepath)
    check_output(['cp', '-r', doc_filepath, new_version], cwd=repo.filepath)
    index_md_filename = join(repo.filepath, 'index.md')
    lines = []
    with open(index_md_filename) as file_:
        lines = file_.readlines()
    # index.md should be at least 5 lines long and have the first bullet
    # (latest) on line 4.
    if len(lines) < 5 or lines[3] != '\n' or not lines[4].startswith('*'):
        raise Exception('index.md has an unexpected format')
    lines[4] = lines[4].replace(' (latest)', '', 1)
    bullet = ('* [v{nv} (latest)]'
              '(http://google.github.io/google-api-nodejs-client'
              '/{nv}/index.html)\n').format(nv=new_version)
    lines.insert(4, bullet)
    with open(index_md_filename, 'w') as file_:
        file_.write(''.join(lines))
    repo.add(['latest', new_version])
    repo.commit(new_version, github_account.name, github_account.email)
    repo.push(branch='gh-pages')
    repo.checkout('master')


def _update_package_json(repo, new_version):
    filename = join(repo.filepath, 'package.json')
    data = {}
    with open(filename) as file_:
        # Note, we use `json.loads` instead of `json.load` here, and
        # `json.dumps` instead of `json.dump` below, because it's easier to
        # just mock `open` than it is to mock `open` and both `json` functions.
        data = json.loads(file_.read())
    data['version'] = new_version
    with open(filename, 'w') as file_:
        file_.write(json.dumps(data, indent=2))


def update(filepath, github_account):
    """Updates the google-api-nodejs-client repository.

    Args:
        filepath (str): the directory to work in.
        github_account (GitHubAccount): the GitHub account to commit with.
    """
    repo = _git.clone_from_github(
        _REPO_PATH, join(filepath, _REPO_NAME), github_account=github_account)
    _install_dependencies(repo)
    added, deleted, updated = _generate_all_clients(repo)
    if not any([added, deleted, updated]):
        return
    _build(repo)
    _run_tests(repo)
    commitmsg = _commit_message.build(added, deleted, updated)
    repo.add(['apis'])
    repo.commit(commitmsg, github_account.name, github_account.email)
    repo.push()


def release(filepath, github_account, npm_account):
    """Releases a new version in the google-api-nodejs-client repository.

    A release consists of:
        1. A Git tag of a new version.
        2. An update to "package.json" and "CHANGELOG.md".
        3. A package pushed to npm.
        4. Generated docs updated on the branch "gh-pages".

    Args:
        filepath (str): the directory to work in.
        github_account (GitHubAccount): the GitHub account to commit with.
    """
    repo = _git.clone_from_github(
        _REPO_PATH, join(filepath, _REPO_NAME), github_account=github_account)
    latest_tag = repo.latest_tag()
    version = _Version(latest_tag)
    authors = repo.authors_since(latest_tag)
    if not authors or not all([x == github_account.email for x in authors]):
        return
    _check_latest_version(latest_tag)
    added, deleted, updated = set(), set(), set()
    status_to_ids = {
        _git.Status.ADDED: added,
        _git.Status.DELETED: deleted,
        _git.Status.UPDATED: updated
    }
    diff_ns = repo.diff_name_status(rev=latest_tag)
    for filename, status in diff_ns:
        match = _SERVICE_FILENAME_RE.match(filename)
        if not match:
            continue
        status_to_ids.get(status, set()).add(match.group(1))
    if deleted:
        version.bump_major()
    else:
        version.bump_minor()
    new_version = str(version)
    _update_package_json(repo, new_version)
    _update_changelog_md(repo, new_version, added, deleted, updated)
    _install_dependencies(repo)
    _build(repo)
    _run_tests(repo)
    repo.commit(new_version, github_account.name, github_account.email)
    repo.tag(new_version)
    repo.push()
    repo.push(tags=True)
    _publish_package(repo, npm_account)
    _update_and_publish_gh_pages(repo, new_version, github_account)
