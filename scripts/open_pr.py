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

def main():
    logging.basicConfig(level=logging.INFO)
    if has_changes():
        print("Git changes detected. Opening pull request ...")
        setup()
        open_pr()
        print("Complete.")
    else:
        print("No git changes. Bailing.")

def has_changes():
    return str(subprocess.run(["git", "status", "-s"], capture_output=True).stdout, "utf-8").strip() != ""

def setup():
    ensure_git_identity()
    ensure_git_fork()

def ensure_git_identity():
    if subprocess.run(["git", "config", "--get", "user.name"], capture_output=True).returncode != 0:
        logging.info("Need to set git user.name.")
        subprocess.run(["git", "config", "--local", "user.name", "Yoshi Automation Bot"])
    if subprocess.run(["git", "config", "--get", "user.email"], capture_output=True).returncode != 0:
        logging.info("Need to set git user.email.")
        subprocess.run(["git", "config", "--local", "user.email", "yoshi-automation@google.com"])

def ensure_git_fork():
    username = str(subprocess.run(["gh", "api", "/user", "--jq=.login"], capture_output=True).stdout, "utf-8").strip()
    subprocess.run(["gh", "repo", "fork", REPO_NAME, "--remote=false", "--clone=false"], check=True)
    fork_repo_name = REPO_NAME.replace("googleapis/", f"{username}/")
    subprocess.run(["gh", "repo", "sync", fork_repo_name], check=True)
    if subprocess.run(["git", "remote", "get-url", REMOTE_NAME], capture_output=True).returncode != 0:
        logging.info(f"Need to create the remote {REMOTE_NAME}.")
        token = os.getenv("GITHUB_TOKEN")
        subprocess.run(["git", "remote", "add", REMOTE_NAME, f"https://{username}:{token}@github.com/{fork_repo_name}.git"], check=True)

def open_pr():
    branch = f"autopr/{uuid.uuid4().hex}"
    timestamp = time.strftime("%Y%m%d-%H%M%S")
    commit_message = f"chore: Automated update of discovery documents at {timestamp}"
    commit_changes(branch, commit_message)
    push_changes(branch)
    pr_number = create_pr(commit_message)
    update_pr(pr_number)

def commit_changes(branch, commit_message):
    logging.info(f"Committing changes to branch {branch}.")
    subprocess.run(["git", "switch", "-c", branch], check=True)
    subprocess.run(["git", "add", "."], check=True)
    subprocess.run(["git", "commit", "-m", commit_message], check=True)

def push_changes(branch):
    logging.info(f"Pushing branch {branch} to remote {REMOTE_NAME}.")
    # This config is set by github actions. Need to undo it temporarily because
    # because otherwise it overrides the auth in the remote url
    result = subprocess.run(["git", "config", "--local", "--get-all", "http.https://github.com/.extraheader", "^AUTHORIZATION:"], capture_output=True)
    existing_auth = str(result.stdout, "utf-8").splitlines()
    if len(existing_auth) > 0:
        logging.info("Note: need to unwind auth header configs for push.")
        subprocess.run(["git", "config", "--local", "--unset-all", "http.https://github.com/.extraheader", "^AUTHORIZATION:"], check=True)
    subprocess.run(["git", "push", "-u", REMOTE_NAME, branch], check=True)
    for auth in existing_auth:
        logging.info("Note: Restoring auth header configs after push.")
        subprocess.run(["git", "config", "--local", "--add", "http.https://github.com/.extraheader", auth], check=True)

def create_pr(commit_message):
    logging.info("Creating pull request.")
    pr_body = "Automatically created by the update_disco script."
    result = subprocess.run(["gh", "pr", "create", "--repo", REPO_NAME, "--title", commit_message, "--body", pr_body], capture_output=True, check=True)
    pr_number = str(result.stdout, "utf-8").splitlines()[-1].split("/")[-1]
    logging.info(f"Pull request number is {pr_number}.")
    for count in range(5):
        if subprocess.run(["gh", "pr", "view", pr_number, "--repo", REPO_NAME, "--json=number"]).returncode == 0:
            logging.info("Confirmed existence of new pull request.")
            break
        logging.info("Couldn't confirm pull request yet ...")
        sleep(2)
    sleep(5)
    return pr_number

def update_pr(pr_number):
    # Use separate approval token to add the automerge label and approve the PR
    approval_token = os.getenv("APPROVAL_GITHUB_TOKEN")
    if approval_token == None:
        logging.info("No approval token provided; skipping automerge")
        return
    main_token = os.getenv("GITHUB_TOKEN")
    os.putenv("GITHUB_TOKEN", approval_token)
    logging.info("Adding automerge label ...")
    subprocess.run(["gh", "issue", "edit", pr_number, "--repo", REPO_NAME, "--add-label", "automerge"])
    approval_message = "Rubber-stamped automated update of discovery documents!"
    logging.info("Approving pull request ...")
    subprocess.run(["gh", "pr", "review", pr_number, "--repo", REPO_NAME, "--approve", "--body", approval_message])
    os.putenv("GITHUB_TOKEN", main_token)
    logging.info("Done with automerge setup")

if __name__ == "__main__":
    main()
