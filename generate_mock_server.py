import argparse
import json
import os
import re
import sys


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


class Generator(object):

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

    _file = sys.stdout

    def __init__(self, root):
        self._root = root
        self._features = root.get('features', [])
        schemas = {}
        for schema in root.get('schemas', {}).itervalues():
            id_ = schema['id']
            schemas[id_] = schema
        self._schemas = schemas

    def set_file(self, file_):
        self._file = file_

    def emit(self, methods):
        w = self._w

        w('import gzip')
        w('import io')
        w('from flask import Flask')
        w('from flask import jsonify')
        w('from flask import request')
        w('')
        w('app = Flask(__name__)')
        w('')
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
        for method in methods.itervalues():
            self._emit_method(method)
            w('')
        w('@app.errorhandler(ApiError)')
        w('def handle_api_error(error):')
        w('    response = jsonify(error.to_dict())')
        w('    response.status_code = error.code')
        w('    return response')
        w('')
        # Got this idea from here:
        # https://github.com/cralston0/gzip-encoding/blob/0b13fcc6381324239cb8ae0712516d90a7fb1ac0/flask/middleware.py
        # TODO: Explain why this middleware is necessary.
        w('@app.before_request')
        w('def handle_gzip():')
        w('    if request.headers.get("content-encoding", "") != "gzip":')
        w('        return')
        w('    file_ = gzip.GzipFile(fileobj=io.BytesIO(request.get_data()))')
        w('    request._cached_data = file_.read()')
        w('')
        w('if __name__ == "__main__":')
        w('    app.run(port=8000)')

    def _emit_method(self, method):
        w = self._w

        path = method['path'].strip('/')
        service_path = self._root['servicePath'].strip('/')
        # The full path is actually "{servicePath}/{path}".
        path = '/'.join([service_path, path]).strip('/')

        # Pull all the braced URL variable names out of the path.
        # Note that we ignore the '+' in multi-segment variable names.
        # Variable names are escaped to prevent naming conflicts.
        # ex: '/foo/{bar}/{+baz}' => ['bar_', 'baz_']
        url_vars = [_esc_var(x) for x in re.findall(r'{\+?([^}]+)}', path)]

        # Convert path into a Flask route.
        # ex: "{+foo}" => "<path:foo>"
        path = re.sub(r'{(\+[^}]+)}', '<path:{}>', path)
        # ex: "{foo}" => "<string:foo>"
        path = re.sub('{([^}]+)}', '<string:{}>', path)
        path = path.format(*url_vars)

        http_verbs = [ method['httpMethod'] ] # ex: [ "POST" ]
        # TODO: For some reason, the Java client library sends PATCH requests
        # as POST requests.
        if http_verbs[0] == 'PATCH':
            http_verbs.append('POST')
        w('@app.route("/{}", methods={})'.format(path, json.dumps(http_verbs)))
        method_name = method['id'].replace('.', '_')
        w('def {}({}):'.format(method_name, ', '.join(url_vars)))

        params = method.get('parameters', {})
        param_order = method.get('parameterOrder', {})
        for name in param_order:
            self._emit_param_assert(name, params[name])

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

        if 'response' in method:
            ref = method['response']['$ref']
            obj = self._gen_type(self._schemas[ref])
            if 'dataWrapper' in self._features:
                obj = {'data': obj}
            # TODO: Explain.
            for key in ['pageToken', 'nextPageToken']:
                obj.pop(key, None)
            w('    return jsonify({})'.format(obj))
        else:
            w('    return jsonify({})')

    def _emit_param_assert(self, name, param):
        w = self._w

        # TODO(saicheems): Because the samples are generated with array fields
        # initialized as empty arrays, client libraries may interpret the value
        # as null. As a result, they may not send anything for the field, which
        # results in a 400 error.
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

    def _w(self, data):
        self._file.write(data + '\n')


def _esc_var(name):
    # Return an identifier that can't conflict with any
    # built-ins/keywords/other vars.
    return name + '_'


def _parse_methods(root, methods=None):
    if methods is None:
        methods = {}
    for method in root.get('methods', {}).itervalues():
        id_ = method['id']
        methods[id_] = method
    for resource in root.get('resources', {}).itervalues():
        _parse_methods(resource, methods)
    return methods


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('file')
    parser.add_argument('--directory', default='mocks')
    args = parser.parse_args()

    root = {}
    with open(args.file) as file_:
        root = json.load(file_)

    methods = _parse_methods(root)
    gen = Generator(root)

    if not os.path.exists(args.directory):
        os.makedirs(args.directory)
    static_dir = os.path.join(args.directory, 'static')
    if not os.path.exists(static_dir):
        os.makedirs(static_dir)

    #root_copy = root.copy()
    #root_copy['rootUrl'] = 'http://localhost:8000/'
    name, version = root['name'], root['version']
    # TODO: Note that the Discovery doc written to the static directory is the
    # same as the one passed in. The passed Discovery doc should already have
    # been modified to point to localhost:8000.
    ddoc_path = os.path.join(static_dir, '{}.{}.json'.format(name, version))
    # Write the Discovery doc to the static directory.
    with open(ddoc_path, 'w') as file_:
        file_.write(json.dumps(root, sort_keys=True, indent=2))
    # Write proxy.html to the static directory.
    # proxy.html is required by the JavaScript client library.
    with open(os.path.join(static_dir, 'proxy.html'), 'w') as file_:
        file_.write(_PROXY_HTML)

    # Verify that all paths are unique. Error if we encounter a conflict.
    paths = {} # Map from reduced method paths to method IDs.
    for id_, method in methods.iteritems():
        # TODO: Check if flatPath is always specified.
        path = method.get('flatPath', method['path']).strip()
        path = re.sub(r'{[\+][^}]*}', '{+}', path)
        path = re.sub(r'{[^\+][^}]*}', '{}', path)
        path = path + ':' + method['httpMethod']
        if path in paths:
            msg = 'method "{}" and "{}" have the same path'
            msg = msg.format(paths[path], id_)
            raise Exception(msg)
        paths[path] = id_

    filename = os.path.join(args.directory,
                            '{}.{}.mock.py'.format(name, version))
    with open(filename, 'w') as file_:
        gen.set_file(file_)
        gen.emit(methods)


if __name__ == '__main__':
    main()
