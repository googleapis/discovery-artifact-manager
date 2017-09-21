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

"""Contains update/release functions for google-api-ruby-client."""

import re

import os
from os.path import join

from tasks import _commit_message, _git
from tasks._check_output import check_output

_REPO_NAME = 'google-api-ruby-client'
_REPO_PATH = 'google/google-api-ruby-client'
# Matches strings like "google-api-client (0.13.6)".
# Not anchored with "^" because `gem search` prints a few leading lines.
_GEM_SEARCH_RE = re.compile(r'google-api-client \((.+)\)$')
# Matches strings like "generated/google/apis/foo_v1.rb".
_SERVICE_FILENAME_RE = re.compile(r'generated/google/apis/([^/]+)\.rb')
# Matches strings like "0.13.6".
_VERSION_RE = re.compile(r'^0\.([0-9]+)\.([0-9]+)$')


class _Version(object):
    """Represents a version of the format "0.1.2"."""

    def __init__(self, latest_tag):
        match = _VERSION_RE.match(latest_tag)
        if not match:
            raise Exception(
                'latest tag does not match the pattern "{}": {}'.format(
                    _VERSION_RE.pattern, latest_tag))
        self.minor_version = int(match.group(1))
        self.patch_version = int(match.group(2))

    def __str__(self):
        return '0.{}.{}'.format(self.minor_version, self.patch_version)

    def bump_minor(self):
        self.minor_version += 1
        self.patch_version = 0

    def bump_patch(self):
        self.patch_version += 1


def _check_latest_version(latest_tag):
    output = check_output(['gem', 'search', '-e', '-r', 'google-api-client'])
    latest_version = _GEM_SEARCH_RE.match(output).group(1)
    if latest_tag != latest_version:
        raise Exception(
            ('latest tag does not match the latest package version on'
             ' RubyGems: {} != {}').format(latest_tag, latest_version))


def _generate_all_clients(repo):
    check_output(['rm', '-rf', 'generated'], cwd=repo.filepath)
    check_output(['./script/generate'], cwd=repo.filepath)
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
        status_to_ids.get(status, set()).add(match.group(1))
    return added, deleted, updated


def _install_dependencies(repo):
    check_output(['bundle', 'install', '--path', 'vendor/bundle'],
                 cwd=repo.filepath)


def _package_and_push_gem(repo, rubygems_account, new_version):
    check_output(['./script/package'], cwd=repo.filepath)
    credentials_filename = os.path.expanduser('~/.gem/credentials')
    with open(credentials_filename, 'w') as file_:
        file_.write('---\n:rubygems_api_key: {}\n'.format(
            rubygems_account.api_key))
    # The credentials file must have permissions of `0600`.
    os.chmod(credentials_filename, 0o600)
    check_output(
        ['gem', 'push', 'pkg/google-api-client-{}.gem'.format(new_version)],
        cwd=repo.filepath)


def _run_tests(repo):
    check_output(['bundle', 'exec', 'rake', 'spec'], cwd=repo.filepath)


def _update_changelog_md(repo, new_version, added, deleted, updated):
    data = '# {}\n'.format(new_version)
    if deleted:
        data += '* Breaking changes:\n'
    for name_version in sorted(deleted):
        data += '  * Deleted `{}`\n'.format(name_version)
    if added or updated:
        data += '* Backwards compatible changes:\n'
    for name_version in sorted(added):
        data += '  * Added `{}`\n'.format(name_version)
    for name_version in sorted(updated):
        data += '  * Updated `{}`\n'.format(name_version)
    data += '\n'
    filename = os.path.join(repo.filepath, 'CHANGELOG.md')
    with open(filename) as file_:
        data = data + file_.read()
    with open(filename, 'w') as file_:
        file_.write(data)


def _update_version_rb(repo, new_version):
    filename = join(repo.filepath, 'lib/google/apis/version.rb')
    data = ''
    with open(filename) as file_:
        data = file_.read()
    data = re.sub(
        'VERSION = \'.+\'', 'VERSION = \'{}\''.format(new_version), data, 1)
    with open(filename) as file_:
        file_.write(data)


def update(filepath, github_account):
    """Updates the google-api-ruby-client repository.

    Args:
        filepath (str): the directory to work in.
        discovery_documents (dict(str, str)): a map of API IDs to Discovery
            document filenames to generate from.
        github_account (GitHubAccount): the GitHub account to commit with.
    """
    repo = _git.clone_from_github(
        _REPO_PATH, join(filepath, _REPO_NAME), github_account=github_account)
    _install_dependencies(repo)
    added, deleted, updated = _generate_all_clients(repo)
    if not any([added, deleted, updated]):
        return
    _run_tests(repo)
    commitmsg = _commit_message.build(added, deleted, updated)
    repo.add(['api_names_out.yaml', 'generated'])
    repo.commit(commitmsg, github_account.name, github_account.email)
    repo.push()


def release(filepath, github_account, rubygems_account, force=False):
    """Releases a new version in the google-api-ruby-client repository.

    A release consists of:
        1. A Git tag of a new version.
        2. An update to "lib/google/apis/version.rb" and "CHANGELOG.md".
        3. A package pushed to RubyGems.

    Args:
        filepath (str): the directory to work in.
        github_account (GitHubAccount): the GitHub account to commit with.
        force (bool, optional): if true, the check that all authors since the
            last tag were `github_account` is ignored.
    """
    repo = _git.clone_from_github(
        _REPO_PATH, join(filepath, _REPO_NAME), github_account=github_account)
    latest_tag = repo.latest_tag()
    version = _Version(latest_tag)
    authors = repo.authors_since(latest_tag)
    if not authors:
        return
    if not force and not all([x == github_account.email for x in authors]):
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
        version.bump_minor()
    else:
        version.bump_patch()
    new_version = str(version)
    _update_version_rb(repo, new_version)
    _update_changelog_md(repo, new_version, added, deleted, updated)
    _install_dependencies(repo)
    _run_tests(repo)
    repo.commit(new_version, github_account.name, github_account.email)
    repo.tag(new_version)
    repo.push()
    repo.push(tags=True)
    _package_and_push_gem(repo, rubygems_account, new_version)
