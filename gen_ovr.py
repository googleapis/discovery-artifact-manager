import argparse
import json
import random
import rstr


def _parse_methods(root, methods=None):
    if methods is None:
        methods = {}
    for method in root.get('methods', {}).itervalues():
        id_ = method['id']
        methods[id_] = method
    for resource in root.get('resources', {}).itervalues():
        _parse_methods(resource, methods)
    return methods


def _gen_fields(method, quote='"'):
    params = method.get('parameters', {})

    fields = {}
    for name, param in params.iteritems():
        if not param.get('required'):
            continue
        pattern = param.get('pattern')
        if not pattern:
            if param.get('type') == 'string' and not param.get('format'):
                default_value = ' '
                if param.get('enum'):
                    default_value = param.get('enum')[0]
                fields[name] = {'defaultValue': quote + default_value + quote}
            # In the Node.js client, required integers must be non-zero.
            # TODO: Has to be int64(0) in Go :(
            #elif param.get('type') == 'integer':
            #    fields[name] = {'defaultValue': 1}
            continue
        if pattern[0] == '^' and pattern[-1] == '$':
            continue
        random.seed(7)
        def_val = rstr.xeger(pattern)
        fields[name] = {'defaultValue': quote + def_val + quote}
    return fields


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('file')
    parser.add_argument('--output')
    args = parser.parse_args()

    root = {}
    with open(args.file) as file_:
        root = json.load(file_)

    methods = _parse_methods(root)

    double_quote = {}
    single_quote = {}
    for id_, method in methods.iteritems():
        fields = _gen_fields(method)
        if not fields:
            continue
        double_quote[id_] = {'fields': fields}
        single_quote[id_] = {'fields': _gen_fields(method, quote='\'')}

    obj = {}
    if double_quote:
        obj['csharp|go|java'] = {'methods': double_quote}
        obj['js|nodejs|php|python|ruby'] = {'methods': single_quote}

    output = ''
    if args.output:
        output = args.output
    else:
        name, version = root['name'], root['version']
        output = '{}.{}.override.json'.format(name, version)
    with open(output, 'w') as file_:
        file_.write(json.dumps(obj, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()
