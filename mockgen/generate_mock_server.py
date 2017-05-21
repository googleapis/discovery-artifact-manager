import argparse
import json
import os
import re

import discoveryutil

_PROXY_HTML = """<!DOCTYPE html>
<html>
<head>
<title></title>
<meta http-equiv="X-UA-Compatible" content="IE=edge" />
<script type="text/javascript">
  window['startup'] = function() {
    googleapis.server.init();
  };
</script>
<script type="text/javascript"
  src="https://apis.google.com/js/googleapis.proxy.js?onload=startup" async
  defer></script>
</head>
<body>
</body>
</html>
"""
"""string: The contents of the proxy.html file.

The JavaScript client library expects this file under
"{rootUrl}/static/proxy.html".
"""


class Generator(object):
    """Generator which produces a mock server from a discovery document."""

    _CAST_FUNC = {
        'any': 'dict',
        'array': 'list',
        'boolean': 'bool',
        'integer': 'int',
        'number': 'float',
        'object': 'dict',
        'string': 'str'
    }

    _INSTANCE = {
        'any': 'object',
        'array': 'list',
        'boolean': 'bool',
        'integer': 'int',
        'number': 'float',
        'object': 'dict',
        'string': 'basestring'
    }

    def __init__(self, root):
        self._root = root

        self._methods = discoveryutil.parse_methods(root)
        # Verify that all paths are unique. Error if we encounter a conflict.
        path_signatures = {} # Map from path signatures to method IDs.
        for id_, method in self._methods.iteritems():
            path_signature = discoveryutil.path_signature(method)
            if path_signature in path_signatures:
                msg = 'method "{}" and "{}" have the same path'
                msg = msg.format(path_signatures[path_signature], id_)
                raise Exception(msg)
            path_signatures[path_signature] = id_

        self._features = root.get('features', [])
        schemas = {}
        for schema in root.get('schemas', {}).itervalues():
            id_ = schema['id']
            schemas[id_] = schema
        self._schemas = schemas

    def emit(self, file_):
        """Emits a mock server.

        Args:
            file_ (:obj:`File`): The file to write to.
        """
        w = self._w(file_)

        # Emit imports and the initialization code for the Flask server.
        w('import gzip')
        w('import io')
        w('from flask import Flask')
        w('from flask import jsonify')
        w('from flask import request')
        w('')
        w('app = Flask(__name__)')
        w('')
        # ApiError represents a "Bad Request" exeception raised by the mock
        # server if parameters of the incomding request is determined to be
        # invalid.
        w('class ApiError(Exception):')
        w('    code = 400')
        w('    message = ""')
        w('')
        w('    def __init__(self, msg):')
        w('        self.message = msg')
        w('')
        w('    def to_dict(self):')
        w("""        return {"error": {
            "code": self.code,
            "message": self.message,
            "details": [],
            "errors": [{
              "message": self.message,
              "domain": "global",
              "reason": "badRequest"
            }]
        }}""")
        w('')
        # Emit handlers for each method.
        for method in self._methods.itervalues():
            self._emit_method(file_, method)
            w('')
        # Emit the error handler for the ApiError exception. If an ApiError is
        # raised, this handler sets the status code and returns the error as a
        # JSON response.
        w('@app.errorhandler(ApiError)')
        w('def handle_api_error(error):')
        w('    response = jsonify(error.to_dict())')
        w('    response.status_code = error.code')
        w('    return response')
        w('')
        # Handle gzipped request bodies. Some client libraries gzip the request
        # body with the expectation that the server decompresses it.
        # Presumably, this would normally be taken care of by a reverse proxy,
        # but we do here manually if the "content-encoding" header is set.
        # This code was adapted from
        # https://github.com/cralston0/gzip-encoding/blob/0b13fcc6381324239cb8ae0712516d90a7fb1ac0/flask/middleware.py
        w('@app.before_request')
        w('def handle_gzip():')
        w('    if request.headers.get("content-encoding", "") != "gzip":')
        w('        return')
        w('    file_ = gzip.GzipFile(fileobj=io.BytesIO(request.get_data()))')
        w('    request._cached_data = file_.read()')
        w('')
        # Run the server on port 8000.
        w('if __name__ == "__main__":')
        w('    app.run(port=8000)')

    def _emit_method(self, file_, method):
        w = self._w(file_)

        # The route is derived from either the "flatPath" or "path" if
        # "flatPath" is not specified.
        path = method.get('flatPath', method['path']).strip('/')
        service_path = self._root['servicePath'].strip('/')
        # The full route is actually "{servicePath}/{path}".
        # ex: "foo/v1" + "bar/{id}" = "foo/v1/bar/{id}"
        path = '/'.join([service_path, path]).strip('/')

        # URL variables in Flask are expected to be inputs to each handler.
        # Thus, the handler for the route
        #     "foo/{fooId}/bar/{barId}"
        # should have the signature
        #     def foo_bar_handler(fooId_, barId_):
        # "path" may contain any number of variables which are braced. They may
        # be multi-segment variables (prepended with a "+", ex: "{+foo}") or
        # single-segment variables (ex: "{foo}"). We pull all variables out of
        # the path and escape them with an underscore to prevent naming
        # conflicts.
        # ex: "/foo/{bar}/{+baz}" => ["bar_", "baz_"]
        url_vars = [_esc_var(x) for x in re.findall(r'{\+?([^}]+)}', path)]

        # Convert path into a Flask route.
        # ex: "{+baz}" => "<path:{}>"
        path = re.sub(r'{(\+[^}]+)}', '<path:{}>', path)
        # ex: "{bar}" => "<string:{}>"
        path = re.sub('{([^}]+)}', '<string:{}>', path)
        # Substitute the variable names back into the path.
        # ex: "foo/<string:bar_>/<path:baz_>"
        path = path.format(*url_vars)

        http_verbs = [ method['httpMethod'] ] # ex: [ "POST" ]
        # We accept the verbs "POST" and "PATCH" for "PATCH" methods because
        # the Java client library sends the "POST" verb for "PATCH" requests.
        if http_verbs[0] == 'PATCH':
            http_verbs.append('POST')

        # Emit the route annotation and method signature.
        w('@app.route("/{}", methods={})'.format(path, json.dumps(http_verbs)))
        method_name = method['id'].replace('.', '_')
        w('def {}({}):'.format(method_name, ', '.join(url_vars)))

        params = method.get('parameters', {})
        # "parameterOrder" contains a list of required parameters.
        param_order = method.get('parameterOrder', {})
        for name in param_order:
            location = params[name]['location']
            # If name is a path variable and not in url_vars then it represents
            # a multi-segment path that was flattened by "flatPath". We don't
            # bother emitting an assert in this case because the reachability
            # of the route is a sufficient check.
            if location == 'path' and (_esc_var(name) not in url_vars):
                continue
            self._emit_param_assert(file_, name, params[name])

        # TODO: It may be useful to reintroduce this check in the future.
        # It verifies that a request body has or hasn't been sent based on
        # whether or not the 'request' field of the method is specified.
        #
        # There are a few issues to sort first:
        # - The Node.js client library will send an empty request body for POST
        #   methods when the 'request' field of the method is not specified.
        #   Some checks fail as a result when an unexpected request body is
        #   observed.
        # - The PHP client library won't send a request body if no fields of
        #   the request body object are set in the calling code. Since no
        #   request body fields are usually set in the samples, this check
        #   can fail erroneously.
        #
        #if 'request' in method:
        #    w('    if not request.data:')
        #    w('        raise ApiError("expected a request body")')
        #    w('    if not isinstance(request.get_json(), dict):')
        #    msg = 'expected the request body to be an instance of \\"dict\\"'
        #    w('        raise ApiError("{}")'.format(msg))
        #else:
        #    w('    if request.data:')
        #    w('        raise ApiError("unexpected request body")')

        # Emit the response.
        if 'response' in method:
            ref = method['response']['$ref']
            obj = self._gen_type(self._schemas[ref])
            # If "dataWrapper" is one of the values in the top-level "features"
            # key, then the client library expects the response to be nested
            # under the key "data".
            if 'dataWrapper' in self._features:
                obj = {'data': obj}
            # We pop the page token key from the response object because page
            # streaming samples in some languages will loop inifnitely if the
            # response contains even a trivial page token value.
            for key in ['pageToken', 'nextPageToken']:
                obj.pop(key, None)
            w('    return jsonify({})'.format(obj))
        else:
            w('    return jsonify({})')

    def _emit_param_assert(self, file_, name, param):
        w = self._w(file_)

        # TODO: Because the samples are generated with array fields initialized
        # as empty arrays, client libraries may interpret the value as null. As
        # a result, they may not send anything for the field, which results in
        # a 400 error.
        # This has to be fixed in the samples by initializing each array with a
        # single entry of its type (ex: [''] instead of []).
        if param.get('repeated'):
            return

        # The only way that 'type' is not a property of param is if it contains
        # a reference to another schema. In that case we assume it's an object.
        type_ = param.get('type', 'object')
        cast_func = self._CAST_FUNC[type_]
        instance = self._INSTANCE[type_]

        location = param['location']
        if location == 'query':
            w('    if "{}" not in request.args:'.format(name))
            msg = 'query parameter \\"{}\\" not found'.format(name)
            w('        raise ApiError("{}")'.format(msg))
            w('    try:')
            cast_func = self._CAST_FUNC[type_]
            w('        {}(request.args.get("{}"))'.format(cast_func, name))
            w('    except:')
            msg = 'expected \\"{}\\" to be an instance of \\"{}\\"'
            msg = msg.format(name, instance)
            w('        raise ApiError("{}")'.format(msg))
        else: # location == 'path'
            # Path params are accessed as variables passed into the function.
            # The variable name is reconstructed here from the param name.
            var = _esc_var(name)
            w('    try:')
            cast_func = self._CAST_FUNC[type_]
            w('        {}({})'.format(cast_func, var))
            w('    except:')
            msg = 'expected \\"{}\\" to be an instance of \\"{}\\"'
            msg = msg.format(name, instance)
            w('        raise ApiError("{}")'.format(msg))

    def _gen_type(self, schema, visited=None):
        if visited is None:
            visited = {}
        if '$ref' in schema:
            param = self._schemas[schema['$ref']]
            return self._gen_type(param, visited)
        type_ = schema['type']
        if type_ == 'any':
            return {}
        if type_ == 'array':
            return [self._gen_type(schema['items'], visited)]
        if type_ == 'boolean':
            return False
        if type_ == 'integer':
            return { 'int32': 42, 'uint32': 42 }.get(schema.get('format', 42))
        if type_ == 'number':
            return { 'double': 42, 'float': 42 }.get(schema.get('format', 42))
        if type_ == 'object':
            obj = {}
            id_ = schema.get('id')
            if id_ in visited:
                return obj
            # Nested objects don't have IDs.
            if id_:
                visited[id_] = True
            for key, val in schema.get('properties', {}).iteritems():
                obj[key] = self._gen_type(val, visited)
            return obj
        if type_ == 'string':
            return {
                'byte': 'foo',
                'date': '1970-01-01',
                'date-time': '1970-01-01T00:00:00-07:00',
                'int64': '42',
                'uint64': '42',
            }.get(schema.get('format'), 'foo')
        raise Exception('unexpected type: {}'.format(type_))

    def _w(self, file_):
        return lambda data: file_.write(data + '\n')


def _esc_var(name):
    return name + '_'


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('file')
    parser.add_argument('--directory', default='mocks')
    args = parser.parse_args()

    root = {}
    with open(args.file) as file_:
        root = json.load(file_)
    gen = Generator(root)
    if not os.path.exists(args.directory):
        os.makedirs(args.directory)
    static_dir = os.path.join(args.directory, 'static')
    if not os.path.exists(static_dir):
        os.makedirs(static_dir)

    name, version = root['name'], root['version']
    # Note that the Discovery doc written to the static directory is the same
    # as the one passed in. The passed Discovery doc should already have been
    # modified to point to localhost:8000.
    ddoc_path = os.path.join(static_dir, '{}.{}.json'.format(name, version))
    # Write the Discovery doc to the static directory.
    with open(ddoc_path, 'w') as file_:
        file_.write(json.dumps(root, sort_keys=True, indent=2))
    # Write proxy.html to the static directory.
    with open(os.path.join(static_dir, 'proxy.html'), 'w') as file_:
        file_.write(_PROXY_HTML)

    filename = '{}.{}.mock.py'.format(name, version)
    filename = os.path.join(args.directory, filename)
    with open(filename, 'w') as file_:
        gen.emit(file_)


if __name__ == '__main__':
    main()
