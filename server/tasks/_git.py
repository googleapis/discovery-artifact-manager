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

"""Implements a utility for cloning and operating on Git repositories."""

import re
from enum import Enum

from tasks._check_output import check_output

# Matches strings like "A\tfoo/bar/baz.txt".
_DIFF_NAME_STATUS_RE = re.compile(r'^(\w)\s+(.*)$')

Status = Enum('Status', 'ADDED DELETED UPDATED UNKNOWN')


def clone_from_github(path, dest, github_account=None):
    """A convenient version `clone` for GitHub hosted repositories.

    Args:
        path (str): the path to the GitHub repository. For example,
            "googleapis/discovery-artifact-manager".
        dest (str): the destination filepath.
        github_account (GitHubAccount, optional): the GitHub account to use to
            encode credentials into the hostname of the constructed repo URL.

    Raises:
        CallError: if the call returns a non-zero return code.

    Returns:
        Repository: the repository.
    """
    hostname = 'github.com'
    if github_account:
        hostname = '{}:{}@{}'.format(github_account.username,
                                     github_account.personal_access_token,
                                     hostname)
    url = 'https://{}/{}'.format(hostname, path)
    return clone(url, dest)


def clone(url, dest):
    """Clones a git Repository.

    Args:
        url (str): the URL of the repository.
        dest (str): the destination filepath.

    Raises:
        CallError: if the call returns a non-zero return code.

    Returns:
        Repository: the repository.
    """
    check_output(['git', 'clone', url, dest])
    return Repository(dest)


class Repository(object):
    """Represents a Git repository."""

    def __init__(self, filepath):
        self.filepath = filepath

    def add(self, filepaths):
        """Add file contents to the index.

        Args:
            paths (list(str)): a list of filepaths to add content from.

        Raises:
            CallError: if the call returns a non-zero return code.
        """
        check_output(['git', 'add', *filepaths], cwd=self.filepath)

    def authors_since(self, rev):
        """Returns a list of emails of the authors of all commits since `rev`.

        Args:
            rev (str): a revision parameter. For example: "d1f3ffe7", or
                "0.13.2".

        Raises:
            CallError: if the call returns a non-zero return code.

        Returns:
            list(str): a list of emails of the authors of all commits since
                `rev`.
        """
        data = check_output(
            ['git', 'log', '{}..HEAD'.format(rev), '--pretty=format:%ae'],
            cwd=self.filepath)
        # Strip whitespace and filter so '' is never returned as an email.
        data = data.strip().split('\n')
        return list(filter(None, data))

    def checkout(self, branch):
        """Switches branches.

        Args:
            branch (str): the name of the branch to checkout.

        Raises:
            CallError: if the call returns a non-zero return code.
        """
        check_output(['git', 'checkout', branch], cwd=self.filepath)

    def checkout_new(self, branch):
        """Create a new branch and switch to it.

        Args:
            branch (str): the name of the branch to checkout.

        Raises:
            CallError: if the call returns a non-zero return code.
        """
        check_output(['git', 'checkout', '-b', branch], cwd=self.filepath)

    def commit(self, message, name, email):
        """Records changes to the repository.

        Args:
            message (str): the commit message.
            name (str): the user's name.
            email (str): the user's email.

        Raises:
            CallError: if the call returns a non-zero return code.
        """
        check_output(
            ['git',
             '-c', 'user.name={}'.format(name),
             '-c', 'user.email={}'.format(email),
             'commit', '-a', '--allow-empty-message', '-m', message],
            cwd=self.filepath)

    def diff_name_status(self, rev=None, staged=True):
        """Returns a list of status, filename pairs of changes from `rev`.

        Args:
            rev (str, optional): a revision parameter. If set, `staged` is
                ignored. For example: "d1f3ffe7", or "0.13.2".
            staged (bool, optional): if true, staged changes are returned.

        Raises:
            CallError: if the call returns a non-zero return code.

        Returns:
            list((str, Status)): a list of filename, Status pairs of changes
                from HEAD.
        """
        args = ['git', 'diff', '--name-status']
        if rev:
            args.append('{}..HEAD'.format(rev))
        elif staged:
            args.append('--staged')
        output = check_output(args, cwd=self.filepath).strip()
        pairs = []
        if not output:
            return pairs
        for line in output.split('\n'):
            match = _DIFF_NAME_STATUS_RE.match(line)
            if not match:
                continue
            status = {
                'A': Status.ADDED,
                'D': Status.DELETED,
                'M': Status.UPDATED
            }.get(match.group(1), Status.UNKNOWN)
            pairs.append((match.group(2), status))
        return pairs

    def latest_tag(self):
        """Returns the latest tag.

        Raises:
            CallError: if the call returns a non-zero return code.

        Returns:
            str: the latest tag.
        """
        data = check_output(
            ['git', 'describe', '--tags', '--abbrev=0'], cwd=self.filepath)
        return data.strip()

    def push(self,
             remote='origin',
             branch='master',
             tags=False,
             nokeycheck=False):
        """Updates remote refs.

        Args:
            remote (str): the remote name.
            branch (str): the branch name.
            tags (bool, optional): if true, only tags are pushed.
            nokeycheck (bool, optional): if true, the
                "--push-option nokeycheck" flag is passed.

        Raises:
            CallError: if the call returns a non-zero return code.
        """
        args = ['git', 'push', remote]
        if tags:
            args.append('--tags')
        else:
            args.append(branch)
        if nokeycheck:
            args.extend(['--push-option', 'nokeycheck'])
        check_output(args, cwd=self.filepath)

    def soft_reset(self, rev):
        """Soft resets current HEAD to `rev`.

        Args:
            rev (str): a revision parameter. For example: "d1f3ffe7",
                or "0.13.2".
            mode (str, optional): the type of reset to perform.

        Raises:
            CallError: if the call returns a non-zero return code.
        """
        args = ['git', 'reset', '--soft', rev]
        check_output(args, cwd=self.filepath)

    def tag(self, name):
        """Creates a tag.

        Args:
            name (str): the name of the tag.

        Raises:
            CallError: if the call returns a non-zero return code.
        """
        check_output(['git', 'tag', name], cwd=self.filepath)
