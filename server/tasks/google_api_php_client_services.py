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

"""Contains update/release functions for google-api-php-client-services."""

import os
import re
from os.path import join
from tempfile import TemporaryDirectory

from tasks import _commit_message, _git
from tasks._check_output import check_output

_REPO_NAME = 'google-api-php-client-services'
_REPO_PATH = 'google/google-api-php-client-services'
# Matches strings like ".../src/Google/Service/BigQuery.php".
_SERVICE_FILENAME_RE = re.compile(r'src/Google/Service/[^/]+\.php$')
# Matches strings like "v0.12".
_VERSION_RE = re.compile(r'^v0\.([0-9]+)$')


class _Version(object):
    """Represents a version of the format "v0.1"."""

    def __init__(self, latest_tag):
        match = _VERSION_RE.match(latest_tag)
        if not match:
            raise Exception(
                'latest tag does not match the pattern "{}": {}'.format(
                    _VERSION_RE.pattern, latest_tag))
        self.minor_version = int(match.group(1))

    def __str__(self):
        return 'v0.{}'.format(self.minor_version)

    def bump_minor(self):
        self.minor_version += 1


def _generate_and_commit_all_clients(repo, venv_filepath, discovery_documents):
    statuses = {}
    for id_, ddoc_filename in discovery_documents.items():
        _generate_client(repo, venv_filepath, ddoc_filename)
        repo.add(['src'])
        diff_ns = repo.diff_name_status()
        # By default, assume no files changed.
        statuses[id_] = None
        if not diff_ns:
            continue
        # If any files changed, the client was updated or added.
        statuses[id_] = _git.Status.UPDATED
        # If the service file is new, the client was added.
        for filename, status in diff_ns:
            match = _SERVICE_FILENAME_RE.match(filename)
            if match and status == _git.Status.ADDED:
                statuses[id_] = _git.Status.ADDED
                break
        # All client commits are soft reset before pushing, so the commit
        # message is left blank and "_" is used for the author name/email.
        repo.commit('', '_', '_')
    added = {k for k, v in statuses.items() if v == _git.Status.ADDED}
    updated = {k for k, v in statuses.items() if v == _git.Status.UPDATED}
    return added, updated


def _generate_client(repo, venv_filepath, ddoc_filename):
    client_filepath = join(repo.filepath, 'src/Google/Service')
    with TemporaryDirectory() as dest_filepath:
        check_output([join(venv_filepath, 'bin/generate_library'),
                      '--input={}'.format(ddoc_filename),
                      '--language=php',
                      '--language_variant=1.2.0',
                      '--output_dir={}'.format(dest_filepath)])
        dirs = os.listdir(dest_filepath)
        client_name = os.path.splitext(dirs[0])[0]  # ex: "BigQuery"
        old_client_filepath = join(client_filepath, client_name)
        old_client_filename = '{}.php'.format(old_client_filepath)
        check_output(['rm', '-rf', old_client_filename, old_client_filepath])
        new_client_filepath = join(dest_filepath, client_name)
        new_client_filename = '{}.php'.format(new_client_filepath)
        check_output(['cp', new_client_filename, old_client_filename])
        check_output(['cp', '-r', new_client_filepath, old_client_filepath])


def _run_tests(repo):
    check_output(['composer', 'update'], cwd=repo.filepath)
    check_output(['vendor/bin/phpunit', '-c', '.'], cwd=repo.filepath)


def update(filepath, discovery_documents, github_account):
    """Updates the google-api-php-client-services repository.

    Args:
        filepath (str): the directory to work in.
        discovery_documents (dict(str, str)): a map of API IDs to Discovery
            document filenames to generate from.
        github_account (GitHubAccount): the GitHub account to commit with.
    """
    repo = _git.clone_from_github(
        _REPO_PATH, join(filepath, _REPO_NAME), github_account=github_account)
    venv_filepath = join(repo.filepath, 'venv')
    check_output(['virtualenv', venv_filepath])
    # The PHP client library generator is published in the
    # "google-apis-client-generator" package.
    check_output([join(venv_filepath, 'bin/pip'),
                  'install',
                  'google-apis-client-generator==1.4.3'])
    added, updated = _generate_and_commit_all_clients(
        repo, venv_filepath, discovery_documents)
    commit_count = len(added) + len(updated)
    if commit_count == 0:
        return
    _run_tests(repo)
    repo.soft_reset('HEAD~{}'.format(commit_count))
    commitmsg = _commit_message.build(added, None, updated)
    repo.commit(commitmsg, github_account.name, github_account.email)
    repo.push()


def release(filepath, github_account):
    """Releases a new version in the google-api-php-client-services repository.

    A release consists of:
        1. A Git tag of a new version.

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
    _run_tests(repo)
    version.bump_minor()
    new_version = str(version)
    repo.tag(new_version)
    repo.push(tags=True)
