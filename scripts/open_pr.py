# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
import os
import subprocess
import time
import uuid


REMOTE_NAME = "yoshi-fork"
REPO_NAME = "googleapis/discovery-artifact-manager"
GIT_USER_NAME = "Yoshi Automation Bot"
GIT_USER_EMAIL = "yoshi-automation@google.com"
COMMIT_MESSAGE = "chore: Automated update of discovery documents"
PULL_REQUEST_BODY = "Automatically created by the update_disco script."
APPROVAL_MESSAGE = "Rubber-stamped automated update of discovery documents!"
MAIN_TOKEN_ENV = "GITHUB_TOKEN"
APPROVAL_TOKEN_ENV = "APPROVAL_GITHUB_TOKEN"


def main():
    """Open and automerge a pull request

    This script checks to see if there are changes in the current clone, and
    opens and optionally automerges a pull request if so. The pull request is
    opened from a fork.

    This script is configured to be used for discovery document updates, and
    should be run immediately after update_disco.py.

    The GITHUB_TOKEN environment variable must be set, providing a token that
    will be used to push the changes and open the pull request. When run from
    a GitHub Action, this should be set to YOSHI_CODE_BOT's token.

    If the APPROVAL_GITHUB_TOKEN environment variable is set, it will be used
    to apply the automerge label to the pull request and approve it. When run
    from a GitHub Action, this should be set to YOSHI_APPROVER's token.
    """
    logging.basicConfig(level=logging.INFO)
    if has_changes():
        print("Git changes detected. Opening pull request ...")
        setup()
        open_pr()
        print("Complete.")
    else:
        print("No git changes. Bailing.")


def has_changes():
    """Determine if there are local changes

    Returns:
        bool -- True if there are local changes, or False otherwise
    """
    return str(subprocess.run(["git", "status", "-s"], capture_output=True).stdout, "utf-8").strip() != ""


def setup():
    """Ensure the right environment for creating a pull request"""
    if os.getenv(MAIN_TOKEN_ENV) == None:
        sys.exit("GITHUB_TOKEN environment variable must be set")
    ensure_git_identity()
    ensure_git_fork()


def ensure_git_identity():
    """Ensure that the git identity (name and email) is set"""
    if subprocess.run(["git", "config", "--get", "user.name"], capture_output=True).returncode != 0:
        logging.info("Need to set git user.name.")
        subprocess.run(["git", "config", "--local", "user.name", GIT_USER_NAME])
    if subprocess.run(["git", "config", "--get", "user.email"], capture_output=True).returncode != 0:
        logging.info("Need to set git user.email.")
        subprocess.run(["git", "config", "--local", "user.email", GIT_USER_EMAIL])


def ensure_git_fork():
    """Ensure that a fork is present and a remote is pointing at it"""
    username = str(subprocess.run(["gh", "api", "/user", "--jq=.login"], capture_output=True).stdout, "utf-8").strip()
    subprocess.run(["gh", "repo", "fork", REPO_NAME, "--remote=false", "--clone=false"], check=True)
    fork_repo_name = REPO_NAME.replace("googleapis/", f"{username}/")
    subprocess.run(["gh", "repo", "sync", fork_repo_name], check=True)
    if subprocess.run(["git", "remote", "get-url", REMOTE_NAME], capture_output=True).returncode != 0:
        logging.info(f"Need to create the remote {REMOTE_NAME}.")
        token = os.getenv(MAIN_TOKEN_ENV)
        subprocess.run(["git", "remote", "add", REMOTE_NAME, f"https://{username}:{token}@github.com/{fork_repo_name}.git"], check=True)


def open_pr():
    """Actually open the pull request"""
    branch = commit_changes()
    push_changes(branch)
    pr_number = create_pr()
    update_pr(pr_number)


def commit_changes(branch):
    """Create a branch and commit the local changes

    Returns:
        str -- The branch name
    """
    branch = f"autopr/{uuid.uuid4().hex}"
    logging.info(f"Committing changes to branch {branch}.")
    subprocess.run(["git", "switch", "-c", branch], check=True)
    subprocess.run(["git", "add", "."], check=True)
    subprocess.run(["git", "commit", "-m", COMMIT_MESSAGE], check=True)
    return branch


def push_changes(branch):
    """Push the branch changes to the fork

    Arguments:
        branch {str} -- The branch name
    """
    logging.info(f"Pushing branch {branch} to remote {REMOTE_NAME}.")
    # This config is set by github actions. Need to undo it temporarily because
    # otherwise it overrides the auth in the remote url.
    result = subprocess.run(["git", "config", "--local", "--get-all", "http.https://github.com/.extraheader", "^AUTHORIZATION:"], capture_output=True)
    existing_auth = str(result.stdout, "utf-8").splitlines()
    if len(existing_auth) > 0:
        logging.info("Note: need to unwind auth header configs for push.")
        subprocess.run(["git", "config", "--local", "--unset-all", "http.https://github.com/.extraheader", "^AUTHORIZATION:"], check=True)
    subprocess.run(["git", "push", "-u", REMOTE_NAME, branch], check=True)
    for auth in existing_auth:
        logging.info("Note: Restoring auth header configs after push.")
        subprocess.run(["git", "config", "--local", "--add", "http.https://github.com/.extraheader", auth], check=True)


def create_pr():
    """Creates a pull request and waits for it to appear in the API

    Returns:
        str -- The pull request number as a string
    """
    logging.info("Creating pull request.")
    result = subprocess.run(["gh", "pr", "create", "--repo", REPO_NAME, "--title", COMMIT_MESSAGE, "--body", PULL_REQUEST_BODY], capture_output=True, check=True)
    pr_number = str(result.stdout, "utf-8").splitlines()[-1].split("/")[-1]
    logging.info(f"Pull request number is {pr_number}.")
    for count in range(5):
        if subprocess.run(["gh", "pr", "view", pr_number, "--repo", REPO_NAME, "--json=number"]).returncode == 0:
            logging.info("Confirmed existence of new pull request.")
            break
        logging.info("Couldn't confirm pull request yet ...")
        time.sleep(count + 1)
    time.sleep(5)
    return pr_number


def update_pr(pr_number):
    """Updates the pull request for automerge

    Arguments:
        pr_number {str} -- The pull request number as a string
    """
    approval_token = os.getenv(APPROVAL_TOKEN_ENV)
    if approval_token == None:
        logging.info("No approval token provided; skipping automerge")
        return
    main_token = os.getenv(MAIN_TOKEN_ENV)
    os.putenv(MAIN_TOKEN_ENV, approval_token)
    logging.info("Adding automerge label ...")
    subprocess.run(["gh", "issue", "edit", pr_number, "--repo", REPO_NAME, "--add-label", "automerge"])
    logging.info("Approving pull request ...")
    subprocess.run(["gh", "pr", "review", pr_number, "--repo", REPO_NAME, "--approve", "--body", APPROVAL_MESSAGE])
    os.putenv(MAIN_TOKEN_ENV, main_token)
    logging.info("Done with automerge setup")


main()
