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
import sys
import time
import uuid
from typing import Optional


REMOTE_NAME = "yoshi-fork"
REPO_NAME = "googleapis/discovery-artifact-manager"
GIT_USER_NAME = "Yoshi Automation Bot"
GIT_USER_EMAIL = "yoshi-automation@google.com"
COMMIT_MESSAGE = "chore: Automated update of discovery documents"
PULL_REQUEST_BODY = "Automatically created by the update_disco script."
APPROVAL_MESSAGE = "Rubber-stamped automated update of discovery documents!"
MAIN_TOKEN_ENV = "GITHUB_TOKEN"
APPROVAL_TOKEN_ENV = "APPROVAL_GITHUB_TOKEN"
AUTOMERGE_DISCOVERY_ENV = "AUTOMERGE_DISCOVERY"


def main() -> None:
    """Open and automerge a pull request

    This script checks to see if there are changes in the current clone, and
    opens and optionally automerges a pull request if so. The pull request is
    opened from a fork.

    This script is configured to be used for discovery document updates, and
    should be run immediately after update_disco.py.

    The GITHUB_TOKEN environment variable, if set, provides a token that will
    be used to push the changes via an https remote, and to open the pull
    request using the gh command line. When run from a GitHub Action, this
    should be set to YOSHI_CODE_BOT's token. If this is not set, the ambient
    ssh credentials will be used to push changes via an ssh remote, and gh will
    have to find credentials elsewhere.

    If the APPROVAL_GITHUB_TOKEN environment variable is set, it will be used
    to apply the automerge label to the pull request and approve it. When run
    from a GitHub Action, this should be set to YOSHI_APPROVER's token. If this
    is not set, the pull request will not be autoapproved or automerged.

    Note that APPROVAL_GITHUB_TOKEN must reference a different user than
    GITHUB_TOKEN, because a user cannot approve their own pull request.
    """
    logging.basicConfig(level=logging.INFO)
    if has_changes():
        print("Git changes detected. Opening pull request ...")
        github_token: Optional[str] = setup()
        open_pr(github_token)
        print("Complete.")
    else:
        print("No git changes. Bailing.")


def has_changes() -> bool:
    """Determine if there are local changes

    Returns:
        bool -- True if there are local changes, or False otherwise
    """
    result: subprocess.CompletedProcess = subprocess.run(
        ["git", "status", "-s"], capture_output=True
    )
    return str(result.stdout, "utf-8").strip() != ""


def setup() -> Optional[str]:
    """Ensure the right environment for creating a pull request

    Returns:
        Optional[str] -- The github token, or None if not provided
    """
    ensure_git_identity()
    github_token: Optional[str] = os.getenv(MAIN_TOKEN_ENV)
    username: str = ensure_github_username()
    fork_repo_name: str = REPO_NAME.replace("googleapis/", f"{username}/")
    ensure_github_fork(fork_repo_name)
    ensure_git_remote(github_token, username, fork_repo_name)
    return github_token


def ensure_git_identity() -> None:
    """Ensure that the git identity (name and email) is set"""
    result: subprocess.CompletedProcess
    result = subprocess.run(
        ["git", "config", "--get", "user.name"], capture_output=True
    )
    if result.returncode != 0:
        logging.info(f"Setting git user.name to {GIT_USER_NAME}")
        subprocess.run(["git", "config", "--local", "user.name", GIT_USER_NAME])
    result = subprocess.run(
        ["git", "config", "--get", "user.email"], capture_output=True
    )
    if result.returncode != 0:
        logging.info(f"Setting git user.email to {GIT_USER_EMAIL}")
        subprocess.run(["git", "config", "--local", "user.email", GIT_USER_EMAIL])


def ensure_github_username() -> str:
    """Get the current GitHub user name, ensuring it exists

    Returns:
        str -- The GitHub user name
    """
    result: subprocess.CompletedProcess = subprocess.run(
        ["gh", "api", "/user", "--jq=.login"], capture_output=True
    )
    username: str = str(result.stdout, "utf-8").strip()
    if len(username) == 0:
        sys.exit(
            "Unable to determine GitHub username; you need to be logged in using gh"
        )
    logging.info(f"Detected GitHub username: {username}")
    return username


def ensure_github_fork(fork_repo_name: str) -> None:
    """Ensure that a GitHub fork is present

    Arguments:
        fork_repo_name {str} -- The expected fork repo name
    """
    logging.info(f"Ensuring a fork exists: {fork_repo_name}")
    subprocess.run(
        ["gh", "repo", "fork", REPO_NAME, "--remote=false", "--clone=false"], check=True
    )
    logging.info(f"Syncing fork: {fork_repo_name}")
    subprocess.run(["gh", "repo", "sync", fork_repo_name], check=True)


def ensure_git_remote(
    github_token: Optional[str], username: str, fork_repo_name: str
) -> None:
    """Ensure that a git remote is present and points at the right place

    Arguments:
        github_token {Optional[str]} -- The github token, if provided
        username {str} -- The GitHub username
        fork_repo_name {str} -- The expected fork repo name
    """
    result: subprocess.CompletedProcess = subprocess.run(
        ["git", "remote", "get-url", REMOTE_NAME], capture_output=True
    )
    remote_url: str
    if result.returncode == 0:
        remote_url = str(result.stdout, "utf-8").strip()
        if fork_repo_name in remote_url:
            logging.info(
                f"Remote {REMOTE_NAME} is already present and seems to reference the fork"
            )
        else:
            sys.exit(
                f"Remote {REMOTE_NAME} has URL {remote_url} which does not seem to reference {fork_repo_name}!"
            )
    else:
        if github_token is None:
            logging.info(f"Creating remote {REMOTE_NAME} using ambient ssh credentials")
            remote_url = f"git@github.com:{fork_repo_name}.git"
        else:
            logging.info(f"Creating remote {REMOTE_NAME} via https using GITHUB_TOKEN")
            remote_url = (
                f"https://{username}:{github_token}@github.com/{fork_repo_name}.git"
            )
        subprocess.run(["git", "remote", "add", REMOTE_NAME, remote_url], check=True)


def open_pr(github_token: Optional[str]) -> None:
    """Actually open the pull request

    Arguments:
        github_token {Optional[str]} -- The github token, if provided
    """
    branch: str = commit_changes()
    push_changes(branch, github_token)
    pr_number: str = create_pr()
    update_pr(pr_number, github_token)


def commit_changes() -> str:
    """Create a branch and commit the local changes

    Returns:
        str -- The branch name
    """
    branch: str = f"autopr/{uuid.uuid4().hex}"
    logging.info(f"Committing changes to branch {branch}.")
    subprocess.run(["git", "switch", "-c", branch], check=True)
    subprocess.run(["git", "add", "."], check=True)
    subprocess.run(["git", "commit", "-m", COMMIT_MESSAGE], check=True)
    return branch


def push_changes(branch: str, github_token: Optional[str]) -> None:
    """Push the branch changes to the fork

    Arguments:
        branch {str} -- The branch name
        github_token {Optional[str]} -- The github token, if provided
    """
    logging.info(f"Pushing branch {branch} to remote {REMOTE_NAME}.")
    existing_auth: list[str] = []
    if github_token is not None:
        # This config is set by github actions. Need to undo it temporarily
        # because otherwise it overrides the auth in the remote url.
        result: subprocess.CompletedProcess = subprocess.run(
            [
                "git",
                "config",
                "--local",
                "--get-all",
                "http.https://github.com/.extraheader",
                "^AUTHORIZATION:",
            ],
            capture_output=True,
        )
        existing_auth = str(result.stdout, "utf-8").splitlines()
        if len(existing_auth) > 0:
            logging.info("Unwinding auth header configs for push.")
            subprocess.run(
                [
                    "git",
                    "config",
                    "--local",
                    "--unset-all",
                    "http.https://github.com/.extraheader",
                    "^AUTHORIZATION:",
                ],
                check=True,
            )
    subprocess.run(["git", "push", "-u", REMOTE_NAME, branch], check=True)
    # Restore old auth config if needed
    for auth in existing_auth:
        logging.info("Note: Restoring auth header configs after push.")
        subprocess.run(
            [
                "git",
                "config",
                "--local",
                "--add",
                "http.https://github.com/.extraheader",
                auth,
            ],
            check=True,
        )


def create_pr() -> str:
    """Creates a pull request and waits for it to appear in the API

    Returns:
        str -- The pull request number as a string
    """
    logging.info("Creating pull request.")
    result: subprocess.CompletedProcess
    result = subprocess.run(
        [
            "gh",
            "pr",
            "create",
            "--repo",
            REPO_NAME,
            "--title",
            COMMIT_MESSAGE,
            "--body",
            PULL_REQUEST_BODY,
        ],
        capture_output=True,
        check=True,
    )
    pr_number: str = str(result.stdout, "utf-8").splitlines()[-1].split("/")[-1]
    logging.info(f"Pull request number is {pr_number}.")
    for count in range(5):
        result = subprocess.run(
            ["gh", "pr", "view", pr_number, "--repo", REPO_NAME, "--json=number"]
        )
        if result.returncode == 0:
            logging.info("Confirmed existence of new pull request.")
            break
        logging.info("Couldn't confirm pull request yet ...")
        time.sleep(count + 1)
    time.sleep(5)
    return pr_number


def update_pr(pr_number: str, github_token: Optional[str]) -> None:
    """Updates the pull request for automerge

    Arguments:
        pr_number {str} -- The pull request number as a string
        github_token {Optional[str]} -- The github token, if provided
    """
    approval_token: Optional[str] = os.getenv(APPROVAL_TOKEN_ENV)
    enable_autoapprove: Optional[bool] = os.getenv(AUTOMERGE_DISCOVERY_ENV) == "true"
    if approval_token is None:
        logging.info("No approval token provided; skipping automerge")
    elif not enable_autoapprove:
        logging.info(f"Autoapproval is not enabled, set {AUTOMERGE_DISCOVERY_ENV} to `true`")
    else:
        os.putenv(MAIN_TOKEN_ENV, approval_token)
        logging.info("Adding automerge label ...")
        subprocess.run(
            [
                "gh",
                "issue",
                "edit",
                pr_number,
                "--repo",
                REPO_NAME,
                "--add-label",
                "automerge",
            ]
        )
        logging.info("Approving pull request ...")
        subprocess.run(
            [
                "gh",
                "pr",
                "review",
                pr_number,
                "--repo",
                REPO_NAME,
                "--approve",
                "--body",
                APPROVAL_MESSAGE,
            ]
        )
        if github_token is None:
            os.unsetenv(MAIN_TOKEN_ENV)
        else:
            os.putenv(MAIN_TOKEN_ENV, github_token)
        logging.info("Done with automerge setup")


if __name__ == "__main__":
    main()
