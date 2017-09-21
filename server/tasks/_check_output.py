# Copyright 2017, Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Contains a convenient wrapper over `subprocess.check_output`."""

from subprocess import PIPE, Popen


class CallError(Exception):
    """Represents a call error."""
    pass


def _call(args, **kwargs):
    proc = Popen(args, stdout=PIPE, stderr=PIPE, **kwargs)
    stdoutdata, stderrdata = proc.communicate()
    return (proc.returncode, stdoutdata.decode('utf-8'),
            stderrdata.decode('utf-8'))


def check_output(args, **kwargs):
    """A convenient version of `subprocess.check_output`.

    Args:
        args (list(str)): a sequence of program arguments.

    Raises:
        CallError: if the call returns a non-zero return code.

    Returns:
        str: the output from `stdout`, as a utf-8 string.
    """
    returncode, stdoutdata, stderrdata = _call(args, **kwargs)
    if returncode:
        message = ''
        if stderrdata:
            message += '\nstderr:\n{}'.format(stderrdata)
        if stdoutdata:
            message += '\nstdout:\n{}'.format(stdoutdata)
        raise CallError(message.rstrip())
    return stdoutdata
