import logging

from flask import abort, Flask, request


app = Flask(__name__)


@app.route('/')
def hello():
    # This header can't be spoofed, see
    # https://cloud.google.com/appengine/docs/flexible/nodejs/scheduling-jobs-with-cron-yaml#securing_urls_for_cron
    if request.headers.get('X-AppEngine-Cron') is None:
        abort(403)
    logging.log(logging.INFO, 'cron: Hello world!')
    return 'Hello world!'


if __name__ == '__main__':
    app.run(host='127.0.0.1', port=8080, debug=True)
