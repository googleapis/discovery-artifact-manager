#!/usr/bin/python2.7
"""Add comments to Json configuration files."""

import json
import re
import sys


# Note that we use $, not \n, because we want to retain the newlines of the
# original, so that line numbers returned by json parse errors can be easily
# mapped to the source.  For similar reasons we don't match whitespace with
# '\s', which would match '\n' and discard newlines before a comment.
COMMENT_PAT = re.compile(r'^[ \t]*#.*$', re.MULTILINE)


def _StripComments(json_string):
  """Strip comments from a json-with-comments string.

  Any line beginning with a pound sign, or with whitespace followed by a pound
  sign, is removed.  Comments are not allowed on the same line as json
  constructs.

  Args:
    json_string: (str) A json string which may contain comments.
  Returns:
    A string without comments.
  """
  return COMMENT_PAT.sub('', json_string)


def Load(fp, **kw):
  """Load json with comments from a file.

  Args:
    fp: (file) A fileish object.
    **kw: (dict) Keyword arguments to pass to the underlying json parser.
  Returns:
    Decoded json data.
  """
  raw = fp.read()
  return Loads(raw, **kw)


def Loads(json_string, **kw):
  """Load json with comments from a string.

  Args:
    json_string: (str|unicode) A string.
    **kw: (dict) Keyword arguments to pass to the underlying json parser.
  Returns:
    Decoded json data.
  """
  stripped = _StripComments(json_string)
  return json.loads(stripped, **kw)


if __name__ == '__main__':
  if len(sys.argv) > 1:
    json_in = open(sys.argv[1])
  else:
    json_in = sys.stdin
  data = Load(json_in)
  json.dump(data, sys.stdout, indent=2)
