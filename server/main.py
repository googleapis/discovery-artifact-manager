import glob
import json
import os
import re
import shlex
import subprocess
from collections import namedtuple, OrderedDict
from datetime import date
from enum import Enum
from tempfile import TemporaryDirectory

from flask import Flask, abort, request
from google.cloud import datastore

app = Flask(__name__)

_DEVNULL = open(os.devnull, 'w')

# Matches strings like "M    apis/foo/v1.ts".
_NODEJS_GIT_DIFF_RE = re.compile(r'(\w)\s+apis/(.+)/(.+)\.ts')

GitHubAccount = namedtuple('GitHubAccount',
                           'name email username personal_access_token')

Repo = Enum('Repo', 'DISCOVERY_ARTIFACT_MANAGER', 'NODEJS', 'PHP')

def _get_github_account():
    """Returns the GitHub account stored in Datastore.

    Returns:
        GitHubAccount: a GitHub account.
    """
    ds = datastore.Client()
    account = list(ds.query(kind='GitHubAccount').fetch())[0]
    return GitHubAccount(account['name'], account['email'],
                         account['username'], account['personal_access_token'])


def _call(cmd, check=False, **kwargs):
    """A wrapper over subprocess.call that splits cmd with shlex.split

    If check is True, then check_call is run instead of call.

    Args:
        cmd (string): A command to run.
        check (bool, optional): If true, check_call is run instead of call.

    Returns:
        int: The returncode of the call.
    """
    func = subprocess.call
    if check:
        func = subprocess.check_call
    return func(shlex.split(cmd), **kwargs)


def _get_repo_url(repo, github_account):
    """Returns an authenticated URL for the given repo.

    Args:
        repo (Repo): a repo.
        github_account (GitHubAccount): the GitHub account to authenticate
            with.

    Returns:
        str: an authenticated URL for the repo.
    """
    path = ''
    if repo == Repo.DISCOVERY_ARTIFACT_MANAGER:
        path = 'googleapis/discovery-artifact-manager'
    if repo == Repo.NODEJS:
        path = 'google/google-api-nodejs-client'
    if repo == Repo.PHP:
        path = 'google/google-api-php-client-services'
    else:
        raise Exception('unknown path: {}'.format(path))
    return 'https://{}:{}@github.com/{}'.format(
        github_account.username, github_account.personal_access_token, path)


def _git_clone(repo, github_account, dest):
    """Clones repo to dest.

    Args:
        repo (Repo): a repo.
        github_account (GitHubAccount): the GitHub account to clone with.
        dest (str): the destination file path.
    """
    url = _get_repo_url(repo, github_account)
    _call('git clone {} {}'.format(url, dest), check=True)


def _git_commit(message, github_account, cwd, check=False):
    """Commits staged changes to the master branch of the repo at cwd.

    Args:
        message (str): the commit message.
        github_account (GitHubAccount): the GitHub account to commit with.
        cwd (str): the directory of the repo.
        check (bool, optional): if true, check_call is run instead of call.

    Returns:
        int: the returncode of the call.
    """
    cmd = ('git -c user.name="{}" -c user.email="{}" commit -a'
           ' --allow-empty-message -m "{}"').format(github_account.username,
                                                    github_account.email,
                                                    message)
    return _call(cmd, check=True, cwd=cwd)


def _git_push(cwd, remote='origin', branch='master', tags=False):
    """Pushes all commits to master of the repo at cwd.

    Args:
        cwd (str): the directory of the repo.
        remote (str, optional): the remote to push to.
        branch (str, optional): the branch to push.
        tags (bool, optional): if true, tags are also pushed.
    """
    cmd = 'git push {} {}'.format(remote, branch)
    _call(cmd, check=True, cwd=cwd, stdout=_DEVNULL, stderr=_DEVNULL)
    if tags:
        cmd = '{} --tags'.format(cmd)
        _call(cmd, check=True, cwd=cwd, stdout=_DEVNULL, stderr=_DEVNULL)


def _verify_git_log(cwd, github_account):
    """Verifies that the git log is valid for releasing a new tag.

    Returns true if there is at least one commit since the last tag and all
    commits since the last tag were authored by `github_account`.

    Args:
        cwd (str): the directory of the repo.
        github_account (GitHubAccount): the GitHub account.

    Returns:
        bool: whether or not a new tag can be released.
    """
    # Get the list of authors for commits since the last tag.
    output = subprocess.check_output(
        shlex.split('git log {}..HEAD --pretty=format:"%ae"'.format(
            latest_tag)),
        cwd=client_lib_dir)
    authors = output.decode('utf-8').strip().split('\n')

    # If there were any commits, and `github_account` was the author of all
    # commits since the last tag, return `True`.
    return bool(output) and all(a == github_account.email for a in authors)


def _build_commitmsg(added, deleted, updated, subject=None):
    """Returns a nice commit message.

    Args:
        added (set): a set of API IDs that have been added.
        deleted (set): a set of API IDs that have been deleted.
        updated (set): a set of API IDs that have been updated.
        subject (str, optional): if provided, replaces the default subject.

    Returns:
        str: a commit message.
    """
    commitmsg = 'Autogenerated update ({})\n'.format(date.today().isoformat())
    if subject:
        commitmsg = '{}\n'.format(subject)
    if added:
        commitmsg += '\nAdd:\n'
        for id_ in sorted(added):
            commitmsg += '- {}\n'.format(id_)
    if deleted:
        commitmsg += '\nDelete:\n'
        for id_ in sorted(deleted):
            commitmsg += '- {}\n'.format(id_)
    if updated:
        commitmsg += '\nUpdate:\n'
        for id_ in sorted(updated):
            commitmsg += '- {}\n'.format(id_)
    return commitmsg


@app.route('/cron/discoveries')
def cron_discoveries():
    # This header can't be spoofed, see
    # https://cloud.google.com/appengine/docs/flexible/python/scheduling-jobs-with-cron-yaml#securing_urls_for_cron
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)

    account = _get_github_account()
    with TemporaryDirectory() as tmp_dir:
        # /tmp/discovery-artifact-manager
        dartman_dir = os.path.join(tmp_dir, 'discovery-artifact-manager')
        _git_clone(Repo.DISCOVERY_ARTIFACT_MANAGER, account, dartman_dir)

        go_dir = os.path.join(tmp_dir, 'go')      # /tmp/go
        os.makedirs(os.path.join(go_dir, 'src'))  # mkdir -p /tmp/go/src
        # ln -s /tmp/discovery-artifact-manager/src \
        #       /tmp/go/src/discovery-artifact-manager
        _call('ln -s {} {}'.format(
            os.path.join(dartman_dir, 'src'),
            os.path.join(go_dir, 'src', 'discovery-artifact-manager')),
              check=True)
        env = os.environ.copy()
        env['GOPATH'] = go_dir
        _call('go run src/main/updatedisco/main.go', check=True,
            cwd=dartman_dir, env=env)

        _call('git add discoveries', cwd=dartman_dir)
        commitmsg = 'Autogenerated Discovery document update'
        returncode = _git_commit(account, commitmsg, cwd)
        # `returncode` is non-zero if there's nothing to commit.
        if not returncode:
            _git_push(dartman_dir)
    return ''


@app.route('/cron/clients/go/update')
def cron_clients_go_update():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)

    account = _get_github_account()
    with TemporaryDirectory() as tmp_dir:
        env = os.environ.copy()
        # /tmp/go
        go_dir = os.path.join(tmp_dir, 'go')
        if not os.path.exists(go_dir):
            os.mkdir(go_dir)
        env['GOPATH'] = go_dir

        # /tmp/go/src/google-api-go-client
        _call('go get -d -t -v google.golang.org/api/...', check=True, env=env)
        client_lib_dir = os.path.join(go_dir, 'src/google.golang.org/api')

        # TODO: Temporary! Remove.
        _call('git fetch origin dartman', check=True, cwd=client_lib_dir)
        _call('git checkout dartman', check=True, cwd=client_lib_dir)

        # Generate all clients.
        _call('make all', check=True, cwd=client_lib_dir, env=env)

        # Run tests.
        _call('go test ./...', check=True, cwd=client_lib_dir, env=env)

        # Stage all changes.
        _call('git add .', check=True, cwd=client_lib_dir)

        # A set of IDs for APIs which have been newly added.
        added = set()
        # A set of IDs for APIs which have been deleted.
        deleted = set()
        # A set of IDs for APIs which have been updated.
        updated = set()

        # `name_version_re` matches strings like
        # "M    foo/v1/foo-gen.go"
        name_version_re = re.compile(r'(\w)\t(.+)/(.+)/.+\.go')
        # Get the names of files that have been changed since the last commit
        # + their status ("A", "D", or "M").
        diff_ns = subprocess.check_output(
            shlex.split('git diff --name-status --staged'), cwd=client_lib_dir)
        # Match for each client and add to the appropriate set.
        for match in name_version_re.finditer(diff_ns.decode('utf-8')):
            status = match.group(1)
            name_version = '{}/{}'.format(match.group(2), match.group(3))
            if status == 'A':
                added.add(name_version)
            elif status == 'D':
                deleted.add(name_version)
            elif status == 'M':
                updated.add(name_version)

        # Bail if no Go files changed. This prevents commits where no actual
        # clients changed, since the generate step also updates the cache of
        # Discovery documents, and the Discovery service does not guarantee
        # that Discovery documents will have a deterministic ordering.
        if not added and not deleted and not updated:
            return ''

        subject = 'all: autogenerated update ({})\n'.format(
            date.today().isoformat())
        commitmsg = _build_commit_msg(added, deleted, updated, subject=subject)
        # A zero return code means there's something to push.
        if _git_commit(commitmsg, account, client_lib_dir) == 0:
            _git_push(client_lib_dir,
                remote='https://code.googlesource.com/_direct/google-api-go-client',
                branch='dartman')
    return ''


@app.route('/cron/clients/nodejs/update')
def cron_clients_nodejs_update():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)

    account = _get_github_account()
    with TemporaryDirectory() as tmp_dir:
        # /tmp/google-api-nodejs-client
        client_lib_dir = os.path.join(tmp_dir, 'google-api-nodejs-client')
        _git_clone(repo.NODEJS, github_account, client_lib_dir)

        # Install dependencies.
        _call('npm install', check=True,
              cwd=client_lib_dir)

        # Generate and build all clients.
        _call('node --max_old_space_size=2000 /usr/bin/npm run generate-apis',
              check=True, cwd=client_lib_dir)
        _call('node --max_old_space_size=2000 /usr/bin/npm run build',
              check=True, cwd=client_lib_dir)

        # Run tests.
        _call('npm run test', check=True, cwd=client_lib_dir)

        # Stage all changes.
        _call('git add .', check=True, cwd=client_lib_dir)

        # A set of IDs for APIs which have been newly added.
        added = set()
        # A set of IDs for APIs which have been deleted.
        deleted = set()
        # A set of IDs for APIs which have been updated.
        updated = set()

        # Get the names of files that have been changed since the last commit
        # + their status ("A", "D", or "M").
        diff_ns = subprocess.check_output(
            shlex.split('git diff --name-status --staged'), cwd=client_lib_dir)
        # Match for each client and add to the appropriate set.
        for match in _NODEJS_GIT_DIFF_RE.finditer(diff_ns.decode('utf-8')):
            status = match.group(1)
            name_version = '{}:{}'.format(match.group(2), match.group(3))
            if status == 'A':
                added.add(name_version)
            elif status == 'D':
                deleted.add(name_version)
            elif status == 'M':
                updated.add(name_version)

        commitmsg = _build_commit_msg(added, deleted, updated)
        # A zero return code means there's something to push.
        if _git_commit(commitmsg, account, client_lib_dir) == 0:
            # Send output to /dev/null so `remote_url` isn't logged.
            _call('git push {}'.format(remote_url), check=True,
                  cwd=client_lib_dir, quiet=True)
    return ''


@app.route('/cron/clients/nodejs/release')
def cron_clients_nodejs_release():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)

    github_account = _get_github_account()
    npm_account = _get_npm_account()
    with TemporaryDirectory() as tmp_dir:
        # /tmp/google-api-nodejs-client
        client_lib_dir = os.path.join(tmp_dir, 'google-api-nodejs-client')
        _git_clone(repo.NODEJS, github_account, client_lib_dir)

        # Get the latest tag.
        output = subprocess.check_output(
            shlex.split('git describe --tags --abbrev=0'),
            cwd=client_lib_dir)
        latest_tag = output.decode('utf-8').strip()

        # Get the latest `googleapis` package version on npm.
        output = subprocess.check_output(
            shlex.split('npm view googleapis version'))
        latest_version = output.decode('utf-8').strip()

        if not _verify_git_log(cwd, github_account):
            return ''

        if latest_tag != latest_version:
            raise Exception(
                ('latest tag does not match the latest package version on npm:'
                 ' {} != {}').format(latest_tag, latest_version))

        # Install dependencies.
        _call('npm install', check=True,
              cwd=client_lib_dir)

        # Build all clients.
        _call('node --max_old_space_size=2000 /usr/bin/npm run build',
              check=True, cwd=client_lib_dir)

        # Run tests.
        _call('npm run test', check=True, cwd=client_lib_dir)

        # `version_re` matches versions like "20.1.0".
        version_re = re.compile(r'^([0-9]+)\.([0-9]+)\.[0-9]+$')
        match = version_re.match(latest_tag)
        if not match:
            raise Exception(
                'latest tag does not match the pattern \'{}\': {}'.format(
                    version_re.pattern, latest_tag))

        major_version = int(match.group(1))
        minor_version = int(match.group(2))

        # Get the names of files that have been changed since the last commit
        # + their status ("A", "D", or "M").
        diff_ns = subprocess.check_output(
            shlex.split(
                'git diff --name-status {}..HEAD --oneline'.format(
                    latest_tag)),
            cwd=client_lib_dir)

        # Get the status for each file.
        statuses = [match.group(1) for match
                    in _NODEJS_GIT_DIFF_RE.finditer(diff_ns.decode('utf-8'))]
        # `changes` is a map of API ID to file status ("A", "D", or "M").
        changes = {}
        for match in _NODEJS_GIT_DIFF_RE.finditer(diff_ns.decode('utf-8')):
            status = match.group(1)
            name_version = '{}:{}'.format(match.group(2), match.group(3))
            changes[name_version] = status

        # If any clients were deleted, increment the major version.
        if 'D' in statuses:
            major_version += 1
        else:  # Otherwise, increment the minor version.
            minor_version += 1

        new_version = '{}.{}.0'.format(major_version, minor_version)

        # Update `package.json` with the new version.
        package_filename = os.path.join(client_lib_dir, 'package.json')
        package_data = None
        with open(package_filename) as file_:
            package_data = file_.read()
        with open(package_filename, 'w') as file_:
            data = json.loads(package_data, object_pairs_hook=OrderedDict)
            data['version'] = new_version
            file_.write(json.dumps(data, indent=2) + '\n')

        # Update `CHANGELOG.md`.
        changelog = '##### {} - {}\n'.format(
            new_version, date.today().strftime("%d %B %Y"))
        if 'D' in statuses:
            changelog += '\n###### Breaking changes\n'
        for name_version in sorted(changes):
            if changes[name_version] == 'D':
                changelog += '- Deleted `{}`\n'.format(name_version)
        if 'A' in statuses or 'M' in statuses:
            changelog += '\n###### Backwards compatible changes\n'
        for name_version in sorted(changes):
            if changes[name_version] == 'A':
                changelog += '- Added `{}`\n'.format(name_version)
        for name_version in sorted(changes):
            if changes[name_version] == 'M':
                changelog += '- Updated `{}`\n'.format(name_version)
        changelog += '\n'
        changelog_filename = os.path.join(client_lib_dir, 'CHANGELOG.md')
        changelog_data = None
        with open(changelog_filename) as file_:
            changelog_data = file_.read()
        with open(changelog_filename, 'w') as file_:
            file_.write(changelog + changelog_data)

        # Commit the changes to `package.json` and `CHANGELOG.md`.
        _git_commit(new_version, account, client_lib_dir)

        _call('git tag {}'.format(new_version), check=True,
              cwd=client_lib_dir)
        _git_push(client_lib_dir, tags=True)

        with open(os.path.expanduser('~/.npmrc'), 'w') as file_:
            file_.write('//registry.npmjs.org/:_authToken={}\n'.format(
                npm_account.auth_token))
        _call('npm publish', check=True, cwd=client_lib_dir)
    return ''


@app.route('/cron/clients/php/update')
def cron_clients_php_update():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)

    account = _get_github_account()
    with TemporaryDirectory() as tmp_dir:
        # /tmp/discovery-artifact-manager
        dartman_dir = os.path.join(tmp_dir, 'discovery-artifact-manager')
        _git_clone(Repo.DISCOVERY_ARTIFACT_MANAGER, account, dartman_dir)
        # /tmp/google-api-php-client-services
        client_lib_dir = os.path.join(tmp_dir, 'google-api-php-client-services')
        _git_clone(Repo.PHP, account, client_lib_dir)

        index_filename = os.path.join(dartman_dir, 'discoveries', 'index.json')
        preferred = {}
        with open(index_filename) as file_:
            root = json.load(file_)
            for api in root['items']:
                preferred[api['id']] = api['preferred']
        # "admin:directory_v1" and "admin:directorytransfer_v1" are incorrectly
        # marked as not preferred.
        preferred['admin:directory_v1'] = True
        preferred['admin:datatransfer_v1'] = True

        # Glob a list of all Discovery documents in discovery-artifact-manager.
        discovery_document_filenames = glob.glob(
            os.path.join(dartman_dir, 'discoveries', '*.json'))
        # Skip index.json.
        discovery_document_filenames = [
            filename for filename in discovery_document_filenames
            if os.path.basename(filename) != 'index.json']

        # /tmp/venv
        venv_dir = os.path.join(tmp_dir, 'venv')
        # Create a Python 2.7 virtualenv.
        _call('virtualenv {}'.format(venv_dir), check=True)
        # Install the Google API client generator.
        _call('{} setup.py install'.format(
            os.path.join(venv_dir, 'bin', 'python')), check=True,
              cwd=os.path.join(dartman_dir, 'google-api-client-generator'))

        # /tmp/google-api-php-client-services/src/Google/Service
        service_dir = os.path.join(client_lib_dir, 'src', 'Google', 'Service')

        # A set of API IDs which have been processed.
        processed = set()
        # A set of IDs for APIs which have been newly added.
        added = set()
        # A set of IDs for APIs which have been updated.
        updated = set()

        # The number of commits that have been made.
        commit_count = 0

        returncode = -1
        for filename in discovery_document_filenames:
            root = {}
            with open(filename) as file_:
                root = json.load(file_)
            id_ = root['id']
            name = root['name']
            version = root['version']

            # The Discovery service is currently returning two APIs with the
            # same ID. In the Discovery directory, both "cloudtrace:v2" and
            # "tracing:v2" point to Discovery documents which are essentially
            # the same. This causes double commits where the "cloudtrace:v2"
            # API is updated first, and a second commit from the "tracing:v2"
            # API is layered on top. Since the generator works off of API name
            # and version, both APIs are generated as "CloudTrace".
            # So, to prevent that corner case, if an API ID has already been
            # processed, skip it.
            if id_ in processed:
                continue
            processed.add(id_)

            # Skip the "discovery" and any non-preferred services.
            if name == 'discovery':
                continue
            if not preferred[id_]:
                continue

            # Generate the service into another temporary directory, so it's
            # possible to decide if any service files should be deleted.
            with TemporaryDirectory() as tmp_dir2:
                # Generate the service into /tmp2/.
                _call(('bin/generate_library'
                       ' --input {}'
                       ' --language php'
                       ' --language_variant 1.2.0'
                       ' --output_dir {}').format(filename, tmp_dir2),
                      check=True, cwd=venv_dir)

                dirs = os.listdir(tmp_dir2)
                # Drop the extension if it's there.
                # ex: "BigQuery" instead of "BigQuery.php".
                service_name = os.path.splitext(dirs[0])[0]
                # Whether or not the service already exists.
                service_exists = os.path.exists(
                    '{}.php'.format(os.path.join(service_dir, service_name)))
                # Delete the original service and service directory.
                # rm -rf /tmp/google-api-php-client-services/src/Google/Service/Foo.php \
                #        /tmp/google-api-php-client-services/src/Google/Service/Foo
                _call('rm -rf {}.php {}'.format(
                    os.path.join(service_dir, service_name),
                    os.path.join(service_dir, service_name)),
                      check=True)
                # Copy the newly generated service back.
                # cp /tmp2/Foo.php /tmp/google-api-php-client-services/src/Google/Service/Foo.php
                _call('cp {}.php {}'.format(
                    os.path.join(tmp_dir2, service_name),
                    service_dir),
                      check=True)
                # cp -r /tmp2/Foo /tmp/google-api-php-client-services/src/Google/Service/Foo
                _call('cp -r {} {}'.format(
                    os.path.join(tmp_dir2, service_name),
                    service_dir),
                      check=True)

            # Stage all changes.
            _call('git add src', check=True, cwd=client_lib_dir)
            # A zero return code means there's something to push.
            if _git_commit('', account, client_lib_dir) == 0:
                commit_count += 1
                if not service_exists:
                    added.add(id_)
                else:
                    updated.add(id_)
                returncode = 0

        # Run tests.
        _call('composer update', check=True, cwd=client_lib_dir)
        _call('vendor/bin/phpunit -c .', check=True, cwd=client_lib_dir)

        # `returncode` is non-zero if there's nothing to commit.
        if not returncode:
            # Reset all the changes so we can combine them into one commit.
            _call('git reset --soft HEAD~{}'.format(commit_count), check=True,
                cwd=client_lib_dir)
            commitmsg = _build_commit_msg(added, None, updated)
            _git_commit(commitmsg, account, client_lib_dir, check=True)
            _git_push(client_lib_dir)
    return ''


@app.route('/cron/clients/php/release')
def cron_clients_php_release():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)

    account = _get_github_account()
    with TemporaryDirectory() as tmp_dir:
        # /tmp/google-api-php-client-services
        client_lib_dir = os.path.join(tmp_dir, 'google-api-php-client-services')
        _git_clone(repo.PHP, account, client_lib_dir)

        # Run tests.
        _call('composer update', check=True, cwd=client_lib_dir)
        _call('vendor/bin/phpunit -c .', check=True, cwd=client_lib_dir)

        # Grab the latest tag.
        output = subprocess.check_output(
            shlex.split('git describe --tags --abbrev=0'),
            cwd=client_lib_dir)
        latest_tag = output.decode('utf-8').strip()

        if not _verify_git_log(cwd, account):
            return ''

        # `version_re` matches versions like "v0.12".
        version_re = re.compile(r'^(v[0-9]+)\.([0-9]+)$')
        match = version_re.match(latest_tag)
        if not match:
            raise Exception(
                'latest tag does not match the pattern \'{}\': {}'.format(
                    version_re.pattern, latest_tag))

        # ex: '12'
        minor_revision = match.group(2)
        # ex: '13'
        new_minor_revision = str(int(minor_revision) + 1)
        # '\1' is substituted with the first group captured by the
        # `version_re` pattern. This replacement defends against the
        # possibility of regressing the major version.
        # ex: 'v0.13'
        new_version = version_re.sub(
            r'\1.{}'.format(new_minor_revision), latest_tag)

        _call('git tag {}'.format(new_version), check=True,
              cwd=client_lib_dir)
        _git_push(cwd, tags=True)
    return ''


if __name__ == '__main__':
    app.run(host='127.0.0.1', port=8080, debug=True)
