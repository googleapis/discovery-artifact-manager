from tempfile import TemporaryDirectory

from flask import Flask, abort, request
from tasks import (accounts,
                   discovery_artifact_manager,
                   google_api_go_client,
                   google_api_nodejs_client,
                   google_api_php_client_services,
                   google_api_ruby_client)

app = Flask(__name__)


@app.route('/cron/discoveries')
def cron_discoveries():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)
    github_account = accounts.get_github_account()
    with TemporaryDirectory() as filepath:
        discovery_artifact_manager.update(filepath, github_account)
    return ''


@app.route('/cron/clients/go/update')
def cron_clients_go_update():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)
    github_account = accounts.get_github_account()
    with TemporaryDirectory() as filepath:
        google_api_go_client.update(filepath, github_account)
    return ''


@app.route('/cron/clients/nodejs/update')
def cron_clients_nodejs_update():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)
    github_account = accounts.get_github_account()
    with TemporaryDirectory() as filepath:
        google_api_nodejs_client.update(filepath, github_account)
    return ''


@app.route('/cron/clients/nodejs/release')
def cron_clients_nodejs_release():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)
    github_account = accounts.get_github_account()
    npm_account = accounts.get_npm_account()
    force = request.args.get('force', default=False, type=bool)
    with TemporaryDirectory() as filepath:
        google_api_nodejs_client.update(
            filepath, github_account, npm_account, force=force)
    return ''


@app.route('/cron/clients/php/update')
def cron_clients_php_update():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)
    github_account = accounts.get_github_account()
    with TemporaryDirectory() as filepath:
        ddocs = discovery_artifact_manager.discovery_documents(
            filepath, preferred=True, skip=['discovery:v1'])
        google_api_php_client_services.update(filepath, ddocs, github_account)
    return ''


@app.route('/cron/clients/php/release')
def cron_clients_php_release():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)
    github_account = accounts.get_github_account()
    force = request.args.get('force', default=False, type=bool)
    with TemporaryDirectory() as filepath:
        google_api_php_client_services.release(
            filepath, github_account, force=force)
    return ''


@app.route('/cron/clients/ruby/update')
def cron_clients_ruby_update():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)
    github_account = accounts.get_github_account()
    with TemporaryDirectory() as filepath:
        google_api_ruby_client.update(filepath, github_account)
    return ''


@app.route('/cron/clients/ruby/release')
def cron_clients_ruby_release():
    if request.headers.get('X-Appengine-Cron') is None:
        abort(403)
    github_account = accounts.get_github_account()
    rubygems_account = accounts.get_rubygems_account()
    force = request.args.get('force', default=False, type=bool)
    with TemporaryDirectory() as filepath:
        google_api_ruby_client.release(
            filepath, github_account, rubygems_account, force=force)
    return ''


if __name__ == '__main__':
    app.run(host='127.0.0.1', port=8080, debug=True)
