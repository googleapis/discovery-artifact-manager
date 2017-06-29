from __future__ import absolute_import
import re
import six


def path_signature(method):
    """Returns the most specific path signature derivable from method.

    Returns path with all braced variable names removed. The braces, and the
    "+" prefix for multi-segment variables, are left in place.  If method
    contains "flatPath", that is used in place of "path". The result is
    qualified with the HTTP verb of the method.

    For example:
        "foo/{fooId}/bar/{+barId}" -> "foo/{}/bar/{+}:POST"

    Args:
        method (dict): A Discovery method.

    Returns:
        string: A path signature.
    """
    path = method.get('flatPath', method['path']).strip()
    path = re.sub(r'{[\+][^}]*}', '{+}', path)
    path = re.sub(r'{[^\+][^}]*}', '{}', path)
    path = '{}:{}'.format(path, method['httpMethod'])
    return path


def parse_methods(root, methods=None):
    """Parses methods from the given Discovery document.

    Args:
        root (dict): A Discovery document. When called recursively, this is a
            resource within a Discovery document.
        methods (dict): A mapping of method ID to Discovery method. Do not set,
            this is used to collect method IDs while recursing.

    Returns:
        dict: A mapping of method ID to method.
    """
    if methods is None:
        methods = {}
    for method in six.itervalues(root.get('methods', {})):
        id_ = method['id']
        methods[id_] = method
    for resource in six.itervalues(root.get('resources', {})):
        parse_methods(resource, methods)
    return methods

