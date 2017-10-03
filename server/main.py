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

"""Flask app which updates various Google-owned repositories."""

from functools import wraps
from tempfile import TemporaryDirectory

from flask import Flask, abort, request

from tasks import (accounts,
                   discovery_artifact_manager,
                   google_api_go_client,
                   google_api_nodejs_client,
                   google_api_php_client_services,
                   google_api_ruby_client)

app = Flask(__name__)


def verify_cron_header(f):
    """Decorator which checks for the header "X-Appengine-Cron"."""
    @wraps(f)
    def wrapper(*args, **kwargs):
        if request.headers.get('X-Appengine-Cron') is None:
            abort(403)
        return f(*args, **kwargs)
    return wrapper


@app.route('/cron/discoveries')
@verify_cron_header
def cron_discoveries():
    github_account = accounts.get_github_account()
    with TemporaryDirectory() as filepath:
        discovery_artifact_manager.update(filepath, github_account)
    return ''


def nodejs_release(filepath, github_account, force):
    """Wrapper over the Node.js release function to standardize parameters."""
    npm_account = accounts.get_npm_account()
    google_api_nodejs_client.release(
        filepath, github_account, npm_account, force=force)


def php_update(filepath, github_account):
    """Wrapper over the PHP release function to standardize parameters."""
    ddocs = discovery_artifact_manager.discovery_documents(
        filepath, preferred=True, skip=['discovery:v1'])
    google_api_php_client_services.update(filepath, github_account, ddocs)


def ruby_release(filepath, github_account, force):
    """Wrapper over the Ruby release function to standardize parameters."""
    rubygems_account = accounts.get_rubygems_account()
    google_api_ruby_client.release(
        filepath, github_account, rubygems_account, force=force)


@app.route('/cron/clients/<string:lang>/update')
@verify_cron_header
def cron_clients_lang_update(lang):
    update = {
        'go': google_api_go_client.update,
        'nodejs': google_api_nodejs_client.update,
        'php': php_update,
        'ruby': google_api_ruby_client.update,
    }.get(lang, None)
    if not update:
        abort(404)
    github_account = accounts.get_github_account()
    with TemporaryDirectory() as filepath:
        update(filepath, github_account)


@app.route('/cron/clients/<string:lang>/release')
@verify_cron_header
def cron_clients_lang_release(lang):
    release = {
        'nodejs': nodejs_release,
        'php': google_api_php_client_services.release,
        'ruby': ruby_release
    }.get(lang, None)
    if not release:
        abort(404)
    github_account = accounts.get_github_account()
    force = request.args.get('force', default=False, type=bool)
    with TemporaryDirectory() as filepath:
        release(filepath, github_account, force=force)


if __name__ == '__main__':
    app.run(host='127.0.0.1', port=8080, debug=True)
