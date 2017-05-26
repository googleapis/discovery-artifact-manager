"""Generates a mock override file from a Discovery document.

The generated file contains override mappings for the default value of all
fields which must have a non-trivial value.
"""

from __future__ import absolute_import
import argparse
import json
import random

import rstr

import discoveryutil
import six


def _gen_fields(method, quote='"'):
    """Returns the override dict for required values for the given method.

    Required fields are assigned a non-zero value that should be accepted by
    all client libraries.

    Args:
        method (dict): A Discovery method.
        quote (string, optional): The symbol to enquote strings with.

    Returns:
        dict: An override mapping for fields in the given method which must
            have a non-trivial value.
    """
    params = method.get('parameters', {})

    fields = {}
    for name, param in six.iteritems(params):
        if not param.get('required'):
            continue
        pattern = param.get('pattern')
        if not pattern:
            is_string = param.get('type') == 'string'
            has_format = bool(param.get('format'))
            is_repeated = param.get('repeated')

            # TODO: Add support for required repeated fields.

            # At the moment it's only useful to override strings which have no
            # format specified (don't bother with bytes or dates). Required
            # strings in several libraries cannot be empty, so we set their
            # value to be a single space.
            if is_string and not has_format and not is_repeated:
                value = 'foo'
                if param.get('enum'):
                    value = param.get('enum')[0]
                fields[name] = {'defaultValue': quote + value + quote}
            # TODO: In the Node.js client, required integers must be non-zero.
            continue
        if pattern[0] == '^' and pattern[-1] == '$':
            continue
        random.seed(7)
        value = rstr.xeger(pattern)
        fields[name] = {'defaultValue': quote + value + quote}
    return fields


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('file')
    parser.add_argument('--output')
    args = parser.parse_args()

    root = {}
    with open(args.file) as file_:
        root = json.load(file_)

    methods = discoveryutil.parse_methods(root)

    # A dict of overrides for languages that use double quotes.
    double_quote = {}
    # A dict of overrides for languages that use single quotes.
    single_quote = {}
    for id_, method in six.iteritems(methods):
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
