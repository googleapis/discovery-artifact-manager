import argparse
import hashlib
import json
import re

def _get_reduced_path(method):
    path = method.get('flatPath', method['path']).strip()
    path = re.sub(r'{[\+][^}]*}', '{+}', path)
    path = re.sub(r'{[^\+][^}]*}', '{}', path)
    path = path + ':' + method['httpMethod']
    return path

def _load_method_paths(root, paths=None):
    if paths is None:
        paths = {}
    for method in root.get('methods', {}).itervalues():
        path = _get_reduced_path(method)
        if path not in paths:
            paths[path] = 0
        paths[path] += 1
    for resource in root.get('resources', {}).itervalues():
        _load_method_paths(resource, paths)
    return set(k for k, v in paths.items() if v > 1)


def _disamb_method_paths(root, paths):
    for method in root.get('methods', {}).itervalues():
        id_ = method['id']
        path = _get_reduced_path(method)
        if path in paths:
            method['path'] = '{}/{}'.format(hashlib.md5(id_).hexdigest(),
                                            method['path'].strip())
    for resource in root.get('resources', {}).itervalues():
        _disamb_method_paths(resource, paths)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('file')
    parser.add_argument('--output')
    args = parser.parse_args()

    root = {}
    with open(args.file) as file_:
        root = json.load(file_)

    root['rootUrl'] = 'http://localhost:8000/'
    paths = _load_method_paths(root)
    _disamb_method_paths(root, paths)

    output = ''
    if args.output:
        output = args.output
    else:
        name, version = root['name'], root['version']
        output = '{}.{}.mock.json'.format(name, version)
    with open(output, 'w') as file_:
        file_.write(json.dumps(root, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()
