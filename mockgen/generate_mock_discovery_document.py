"""Generates a mock Discovery document from a Discovery document.

The generated document points to "http://localhost:8000" and contains no
ambiguous method paths (each method is guaranteed to point to a unique path).
"""

import argparse
import hashlib
import json

import discoveryutil


def _disambiguate_method_paths(root, seen=None):
    """Disambiguates method paths in-place in the given Discovery document.

    Method paths are considered to be conflicting if two methods with the same
    HTTP verb point to the same path.

    For example, the paths
        ("foo/{+bar}", "POST") and ("foo/{+bar}", "GET")
    do not conflict.

    The paths
        ("foo/{+bar}", "POST") and ("foo/{+baz}", "POST")
    do conflict.

    Args:
        root (dict): A Discovery document.
        seen (set, optional): The set of paths which has already been
            encountered. Do not set.
    """
    if seen is None:
        seen = set()
    for method in root.get('methods', {}).itervalues():
        id_ = method['id']
        path_signature = discoveryutil.path_signature(method)
        if path_signature not in seen:
            seen.add(path_signature)
        else:
            hash_ = hashlib.md5(id_).hexdigest()
            method['path'] = '{}/{}'.format(hash_, method['path'].strip())
            print method['path']
    for resource in root.get('resources', {}).itervalues():
        _disambiguate_method_paths(resource, seen)


def _main():
    parser = argparse.ArgumentParser()
    parser.add_argument('file')
    parser.add_argument('--output')
    args = parser.parse_args()

    root = {}
    with open(args.file) as file_:
        root = json.load(file_)

    output = ''
    if args.output:
        output = args.output
    else:
        name, version = root['name'], root['version']
        output = '{}.{}.mock.json'.format(name, version)

    root['rootUrl'] = 'http://localhost:8000/'
    _disambiguate_method_paths(root)

    with open(output, 'w') as file_:
        file_.write(json.dumps(root, indent=2, sort_keys=True))


if __name__ == '__main__':
    _main()
