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
import os
from tempfile import TemporaryDirectory

import pytest

from tasks._check_output import CallError, check_output


def test_check_output():
    stdoutdata = check_output(['echo', 'hello world'])
    assert stdoutdata == 'hello world\n'


def test_check_output_error_stdout():
    program = 'print("stdout stuff");quit(1)'
    with pytest.raises(CallError) as excinfo:
        check_output(['python3', '-c', program])
    assert str(excinfo.value) == '\nstdout:\nstdout stuff'


def test_check_output_error_stderr():
    program = ('import sys;'
               'print("stderr stuff", file=sys.stderr);'
               'quit(1)')
    with pytest.raises(CallError) as excinfo:
        check_output(['python3', '-c', program])
    assert str(excinfo.value) == '\nstderr:\nstderr stuff'


def test_check_output_error_stdout_stderr():
    program = ('import sys;'
               'print("stdout stuff");'
               'print("stderr stuff", file=sys.stderr);'
               'quit(1)')
    with pytest.raises(CallError) as excinfo:
        check_output(['python3', '-c', program])
    assert str(excinfo.value) == ('\nstderr:\n'
                                  'stderr stuff\n'
                                  '\n'
                                  'stdout:\n'
                                  'stdout stuff')


def test_check_output_env():
    env = os.environ.copy()
    env.pop('TESTVAR123', None)
    program = 'import os;print(os.environ["TESTVAR123"]);'
    with pytest.raises(CallError):
        check_output(['python3', '-c', program])
    env['TESTVAR123'] = 'TEST'
    stdoutdata = check_output(['python3', '-c', program], env=env)
    assert stdoutdata == 'TEST\n'


def test_check_output_cwd():
    with TemporaryDirectory() as filepath:
        with open(os.path.join(filepath, 'test'), 'w') as file_:
            file_.write('hello world')
        stdoutdata = check_output(['cat', 'test'], cwd=filepath)
        assert stdoutdata == 'hello world'
