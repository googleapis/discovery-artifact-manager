from __future__ import print_function
import argparse
import collections
import glob
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import time

_DEVNULL = open(os.devnull, 'w')

_CSHARP = 'csharp'
_GO = 'go'
_JAVA = 'java'
_NODEJS = 'nodejs'
_PHP = 'php'
_PYTHON = 'python'
_RUBY = 'ruby'

_LANGUAGES = [_CSHARP, _GO, _JAVA, _NODEJS, _PHP, _PYTHON, _RUBY]

_GAPIC_YAML_FILENAMES = {
    _CSHARP: ('toolkit/src/main/resources/com/google/api/codegen/csharp'
              '/csharp_discovery.yaml'),
    _GO: ('toolkit/src/main/resources/com/google/api/codegen/go'
          '/go_discovery.yaml'),
    _JAVA: ('toolkit/src/main/resources/com/google/api/codegen/java'
            '/java_discovery.yaml'),
    _NODEJS: ('toolkit/src/main/resources/com/google/api/codegen/nodejs'
              '/nodejs_discovery.yaml'),
    _PHP: ('toolkit/src/main/resources/com/google/api/codegen/php'
           '/php_discovery.yaml'),
    _PYTHON: ('toolkit/src/main/resources/com/google/api/codegen/py'
              '/python_discovery.yaml'),
    _RUBY: ('toolkit/src/main/resources/com/google/api/codegen/ruby'
             '/ruby_discovery.yaml')
}


_POM_XML = """<project xmlns="http://maven.apache.org/POM/4.0.0"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
                      http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>

  <groupId>com.google.test</groupId>
  <artifactId>app</artifactId>
  <version>1.0</version>

  <packaging>jar</packaging>

  <dependencies>
    <dependency>
      <groupId>com.google.api-client</groupId>
      <artifactId>google-api-client</artifactId>
      <version>1.22.0</version>
    </dependency>
  </dependencies>

  <build>
    <plugins>
      <plugin>
        <artifactId>maven-assembly-plugin</artifactId>
        <configuration>
          <descriptorRefs>
            <descriptorRef>jar-with-dependencies</descriptorRef>
          </descriptorRefs>
        </configuration>
      </plugin>
    </plugins>
  </build>
</project>
"""


Context = collections.namedtuple('Context', ('discovery_document_filename'
                                             ' override_filenames id_ name'
                                             ' version revision'))

SampleCommand = collections.namedtuple('SampleCommand', 'id_ command cwd')


def _parse_method_id_from_sample_filename(sample_filename):
    # "foo/foo.bar.get.frag.go" -> "foo.bar.get.frag.go"
    name = os.path.basename(sample_filename)
    # "foo.bar.get.frag.go" -> "foo.bar.get"
    return name.rsplit('.', 2)[0]


def _make_lib_dir(test_dir):
    lib_dir = os.path.join(test_dir, 'lib')
    if not os.path.exists(lib_dir):
        os.makedirs(lib_dir)
    return lib_dir


def _make_lib_google_api_client_generator(test_dir):
    lib_dir = _make_lib_dir(test_dir)
    client_generator_dir = os.path.join(lib_dir, 'google-api-client-generator')
    if not os.path.exists(client_generator_dir):
        cmd = 'cp -r google-api-client-generator {}'.format(lib_dir)
        subprocess.call(shlex.split(cmd))
        cmd = 'virtualenv venv'
        subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
        cmd = 'venv/bin/python setup.py install'
        subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
    return client_generator_dir


def _make_lib_venv(test_dir):
    venv_dir = os.path.join(_make_lib_dir(test_dir), 'venv')
    if not os.path.exists(venv_dir):
        cmd = 'virtualenv {}'.format(venv_dir)
        subprocess.call(shlex.split(cmd))

        env = os.environ.copy()
        env['PATH'] = '{}:{}'.format(os.path.join(venv_dir, 'bin'),
                                     env['PATH'])
        cmd = 'python setup.py install'
        subprocess.call(shlex.split(cmd), cwd='mockgen', env=env)
        cmd = 'pip install flask google-api-python-client'
        subprocess.call(shlex.split(cmd), env=env)
    return venv_dir


def _make_src_dir(test_dir, name, version, language=None):
    src_dir = os.path.join(test_dir, 'src', name, version)
    if language:
        src_dir = os.path.join(src_dir, language)
    if not os.path.exists(src_dir):
        os.makedirs(src_dir)
    return src_dir


def _generate_overrides(test_dir, discovery_document_filename, name, version):
    # TODO: Want to set up a virtualenv in test probably.
    src_dir = _make_src_dir(test_dir, name, version)
    filenames = []
    value_override_filename = os.path.join(src_dir, 'override1.json')
    filenames.append(value_override_filename)

    venv_dir = _make_lib_venv(test_dir)
    env = os.environ.copy()
    env['PATH'] = '{}:{}'.format(os.path.join(venv_dir, 'bin'), env['PATH'])

    cmd = 'generate_mock_value_override {} --output {}'
    cmd = cmd.format(discovery_document_filename,
                     value_override_filename)
    subprocess.call(shlex.split(cmd), env=env)

    suffix = 2
    override_filename = '{}.override.json'.format(
            os.path.splitext(discovery_document_filename)[0])
    if os.path.isfile(override_filename):
        filename = os.path.join(src_dir, 'override2.json')
        filenames.append(filename)
        shutil.copy2(original_override_filename, filename)
        suffix += 1

    misc_override_filename = 'override{}.json'.format(suffix)
    misc_override_filename = os.path.join(src_dir, misc_override_filename)
    filenames.append(misc_override_filename)
    with open(misc_override_filename, 'w') as file_:
        url = 'http://localhost:8000/static/{}.{}.json'
        url = url.format(name, version)
        override = {
            'csharp|go|java|js|nodejs|php|python|ruby': {
                'authType': 'NONE'
            },
            'js|python': {
                'discoveryDocUrl': url
            }
        }
        file_.write(json.dumps(override, indent=2, sort_keys=True))
    return filenames


def _generate_samples(ctx, language, ruby_names_file=None):
    temp_dir = tempfile.mkdtemp()
    cmd = ('java -jar discoGen-0.0.5.jar'
           ' --discovery_doc {}'
           ' --gapic_yaml {}'
           ' --overrides {}'
           ' --output {}')
    cmd = cmd.format(ctx.discovery_document_filename,
                     _GAPIC_YAML_FILENAMES[language],
                     ','.join(ctx.override_filenames), temp_dir)
    if ruby_names_file:
        cmd = cmd + ' --ruby_names_file {}'.format(ruby_names_file)
    subprocess.call(shlex.split(cmd))
    autogen_src_dir = os.path.join(temp_dir, 'autogenerated', ctx.name,
                                   ctx.version, ctx.revision)
    ext = {
        _CSHARP: 'cs', _GO: 'go', _JAVA: 'java', _NODEJS: 'njs',
        _PHP: 'php', _PYTHON: 'py', 'ruby': 'rb'
    }[language]
    sample_filenames = glob.glob(os.path.join(autogen_src_dir,
                                              '*.{}'.format(ext)))
    return sample_filenames


def _load_csharp(test_dir, ctxs):
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-dotnet-client')

    cmd = 'git clone https://github.com/google/google-api-dotnet-client'
    subprocess.call(shlex.split(cmd), cwd=lib_dir)
    cmd = 'git reset --hard v1.26.2'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)

    discovery_json_dir = os.path.join(client_lib_dir, 'DiscoveryJson')
    shutil.rmtree(discovery_json_dir)
    os.makedirs(discovery_json_dir)

    for ctx in ctxs:
        shutil.copy(ctx.discovery_document_filename, discovery_json_dir)

    client_generator_dir = os.path.join(client_lib_dir, 'ClientGenerator')
    cmd = 'virtualenv venv'
    subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
    cmd = 'venv/bin/python setup.py install'
    subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
    venv_bin_dir = os.path.join(client_generator_dir, 'venv', 'bin')
    env = os.environ.copy()
    env['PATH'] = '{}:{}'.format(venv_bin_dir, env['PATH'])
    cmd = 'bash BuildGenerated.sh --onlygenerate'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir, env=env)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _CSHARP)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _CSHARP)
        cmd = 'dotnet new sln -n test'
        subprocess.call(shlex.split(cmd), cwd=src_dir)

        csproj_filenames = []
        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            # .../csharp/foo.bar.get/
            project_dir = os.path.join(src_dir, method_id)
            if not os.path.exists(project_dir):
                os.makedirs(project_dir)
            # .../csharp/foo.bar.get/Program.cs
            shutil.copy(filename, os.path.join(project_dir, 'Program.cs'))
            csproj_filename = '{}.csproj'.format(method_id)
            csproj_filename = os.path.join(project_dir, csproj_filename)
            csproj_filenames.append(csproj_filename)

            with open(csproj_filename, 'w') as file_:
                file_.write("""<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <ProjectReference Include="{}" />
  </ItemGroup>
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>netcoreapp1.0</TargetFramework>
  </PropertyGroup>
</Project>
""".format(os.path.join(client_lib_dir, 'Src', 'Generated', '*', '*.csproj')))

            dll_filename = os.path.join(project_dir, 'bin', 'Debug',
                                        'netcoreapp1.0',
                                        '{}.dll'.format(method_id))
            cmd = 'dotnet {}'.format(dll_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, None))

        cmd = 'dotnet sln test.sln add {}'.format(' '.join(csproj_filenames))
        subprocess.call(shlex.split(cmd), cwd=src_dir)
        cmd = 'dotnet restore'
        subprocess.call(shlex.split(cmd), cwd=src_dir)
        cmd = 'dotnet msbuild /m'
        subprocess.call(shlex.split(cmd), cwd=src_dir)

    return sample_cmds


def _load_go(test_dir, ctxs):
    lib_dir = _make_lib_dir(test_dir)
    go_dir = os.path.join(lib_dir, 'go')
    if not os.path.exists(go_dir):
        os.makedirs(go_dir)
    env = os.environ.copy()
    env['GOPATH'] = go_dir
    cmd = 'go get -v google.golang.org/api/google-api-go-generator'
    subprocess.call(shlex.split(cmd), env=env)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _GO)
        sample_cmds[ctx.id_] = []

        version = ctx.version
        if version == 'alpha' or version == 'beta':
            version = 'v0.' + version
        match = re.match(r'^(.+)_(v[\d\.]+)$', version)
        if match:
            version = '{}/{}'.format(match.group(1), match.group(2))
        cmd = '{} --api_json_file {}'.format(
                os.path.join(go_dir, 'bin', 'google-api-go-generator'),
                ctx.discovery_document_filename)
        subprocess.call(shlex.split(cmd), env=env)

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _GO)
        bin_dir = os.path.join(src_dir, 'bin')
        env['GOBIN'] = bin_dir
        cmd = 'ln -s {} {}'
        cmd = cmd.format(src_dir, os.path.join(go_dir, 'src', ctx.id_))
        subprocess.call(shlex.split(cmd))

        sample_cmds[ctx.id_] = []
        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            package_dir = os.path.join(src_dir, method_id)
            if not os.path.exists(package_dir):
                os.makedirs(package_dir)
            new_filename = os.path.join(package_dir, '{}.go'.format(method_id))
            shutil.copy(filename, new_filename)
            cmd = os.path.join(bin_dir, method_id)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, None))

        cmd = 'go get -v ./...'
        subprocess.call(shlex.split(cmd), cwd=src_dir, env=env)

    return sample_cmds


def _load_java(test_dir, ctxs):
    lib_dir = _make_lib_dir(test_dir)
    client_generator_dir = _make_lib_google_api_client_generator(test_dir)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _JAVA)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _JAVA)
        mvn_src_dir = os.path.join(src_dir, 'src', 'main', 'java')
        cmd = ('venv/bin/python src/googleapis/codegen/generate_library.py'
               ' --input {}'
               ' --language java'
               ' --package_path api/services'
               ' --output_dir {}')
        cmd = cmd.format(ctx.discovery_document_filename, mvn_src_dir)
        subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
        with open(os.path.join(src_dir, 'pom.xml'), 'w') as file_:
            file_.write(_POM_XML)

        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            package_dir = os.path.join(mvn_src_dir, method_id)
            if not os.path.exists(package_dir):
                os.makedirs(package_dir)

            sample_class_name = ''
            sample_content = ''
            with open(filename) as file_:
                sample_content = file_.read()
                match = re.search(r'class\s+(\w+)', sample_content)
                sample_class_name = match.group(1)
            new_filename = os.path.join(package_dir,
                                        '{}.java'.format(sample_class_name))
            with open(new_filename, 'w') as file_:
                file_.write('package {};\n'.format(method_id))
                file_.write(sample_content)
            jar_filename = 'app-1.0-jar-with-dependencies.jar'
            jar_filename = os.path.join(src_dir, 'target', jar_filename)
            cmd = 'java -cp {} {}.{}'.format(jar_filename, method_id,
                                             sample_class_name)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, None))

        cmd = 'mvn package assembly:single'
        subprocess.call(shlex.split(cmd), cwd=src_dir)

    return sample_cmds


def _load_nodejs(test_dir, ctxs):
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-nodejs-client')

    cmd = ('git clone --depth 1 '
           'https://github.com/google/google-api-nodejs-client')
    subprocess.call(shlex.split(cmd), cwd=lib_dir)
    cmd = 'npm install'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)
    cmd = 'npm run build-tools'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)
    cmd = 'rm -rf apis'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)

    for ctx in ctxs:
        cmd = 'node scripts/generate {}'
        cmd = cmd.format(ctx.discovery_document_filename)
        subprocess.call(shlex.split(cmd), cwd=client_lib_dir)

    cmd = 'npm run build'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _NODEJS)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _NODEJS)
        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            new_filename = '{}.js'.format(method_id)
            shutil.copy(filename, os.path.join(src_dir, new_filename))
            cmd = 'node {}'.format(new_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, src_dir))

        cmd = 'npm install {}'.format(client_lib_dir)
        subprocess.call(shlex.split(cmd), cwd=src_dir)

    return sample_cmds


def _load_php(test_dir, ctxs):
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-php-client-services')
    cmd = ('git clone --depth 1'
           ' https://github.com/google/google-api-php-client-services')
    subprocess.call(shlex.split(cmd), cwd=lib_dir)
    client_generator_dir = _make_lib_google_api_client_generator(test_dir)

    shutil.rmtree(os.path.join(client_lib_dir, 'src'))

    for ctx in ctxs:
        cmd = ('venv/bin/python src/googleapis/codegen/generate_library.py'
               ' --input {}'
               ' --language php'
               ' --language_variant 1.2.0'
               ' --output_dir {}')
        cmd = cmd.format(
                ctx.discovery_document_filename,
                os.path.join(client_lib_dir, 'src', 'Google', 'Service'))
        subprocess.call(shlex.split(cmd), cwd=client_generator_dir)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _PHP)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _PHP)
        with open(os.path.join(src_dir, 'composer.json'), 'w') as file_:
            file_.write("""{{
  "repositories": [
    {{
      "type": "path",
      "url": "{}"
    }}
  ],
  "require": {{
    "google/apiclient": "*",
    "google/apiclient-services": "dev-master"
  }}
}}
""".format(client_lib_dir))

        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            new_filename = '{}.php'.format(method_id)
            shutil.copy(filename, os.path.join(src_dir, new_filename))
            cmd = 'php {}'.format(new_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, src_dir))

        cmd = 'composer update'
        subprocess.call(shlex.split(cmd), cwd=src_dir)

    return sample_cmds


def _load_python(test_dir, ctxs):
    venv_dir = _make_lib_venv(test_dir)
    cmd = '{} install google-api-python-client'
    cmd = cmd.format(os.path.join(venv_dir, 'bin', 'pip'))
    subprocess.call(shlex.split(cmd))

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _PYTHON)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _PYTHON)

        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            new_filename = '{}.py'.format(method_id)
            shutil.copy(filename, os.path.join(src_dir, new_filename))
            cmd = 'python {}'.format(new_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, src_dir))

    return sample_cmds


def _load_ruby(test_dir, ctxs):
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-ruby-client')
    cmd = ('git clone --depth 1'
           ' https://github.com/google/google-api-ruby-client')
    subprocess.call(shlex.split(cmd), cwd=lib_dir)
    cmd = 'bundle install --path vendor/bundle'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)

    shutil.rmtree(os.path.join(client_lib_dir, 'generated'))
    cmd = 'git checkout generated/google/apis/discovery_v1'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)
    cmd = 'git checkout generated/google/apis/discovery_v1.rb'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)

    discovery_document_filenames = []
    for ctx in ctxs:
        discovery_document_filenames.append(ctx.discovery_document_filename)

    names_filename = os.path.join(client_lib_dir, 'api_names.yaml')
    os.remove(names_filename)
    subprocess.call(shlex.split('touch {}'.format(names_filename)))
    cmd = ('bundle exec bin/generate-api gen generated --file {}'
           ' --names_out {}')
    cmd = cmd.format(' '.join(discovery_document_filenames), names_filename)
    proc = subprocess.Popen(shlex.split(cmd), cwd=client_lib_dir,
                            stdin=subprocess.PIPE)
    proc.communicate(input='a')
    proc.wait()

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _RUBY,
                                             ruby_names_file=names_filename)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _RUBY)

        with open(os.path.join(src_dir, 'Gemfile'), 'w') as file_:
            file_.write('source \'https://rubygems.org\'\n')
            line = 'gem \'google-api-client\', :path => \'{}\'\n'
            line = line.format(client_lib_dir)
            file_.write(line)

        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            new_filename = '{}.rb'.format(method_id)
            shutil.copy(filename, os.path.join(src_dir, new_filename))
            cmd = 'bundle exec ruby {}'.format(new_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, src_dir))

        cmd = 'bundle install --path vendor/bundle'
        subprocess.call(shlex.split(cmd), cwd=src_dir)

    return sample_cmds


_LOAD_FUNCS = {
    _CSHARP: _load_csharp,
    _GO: _load_go,
    _JAVA: _load_java,
    _NODEJS: _load_nodejs,
    _PHP: _load_php,
    _PYTHON: _load_python,
    _RUBY: _load_ruby
}


def _run(discovery_document_filenames, languages):
    test_dir = os.path.abspath('test/{}'.format(int(time.time())))
    if not os.path.exists(test_dir):
        os.makedirs(test_dir)

    venv_dir = (_make_lib_venv(test_dir))
    env = os.environ.copy()
    env['PATH'] = '{}:{}'.format(os.path.join(venv_dir, 'bin'), env['PATH'])

    ids = []
    ctxs = []
    for filename in discovery_document_filenames:
        root = {}
        with open(filename) as file_:
            root = json.load(file_)

        id_ = root['id']
        if id_ in ids:
            raise Exception('duplicate file for {}: {}'.format(id_, filename))
        ids.append(id_)

        name = root['name']
        version = root['version']
        revision = root['revision']

        src_dir = _make_src_dir(test_dir, name, version)

        filename2 = '{}.{}.json'.format(name, version)
        filename2 = os.path.join(src_dir, filename2)
        cmd = ('generate_mock_discovery_document {} --output {}')
        cmd = cmd.format(filename, filename2)
        subprocess.call(shlex.split(cmd), env=env)

        cmd = 'generate_mock_server {} --directory {}'
        cmd = cmd.format(filename2, src_dir)
        subprocess.call(shlex.split(cmd), env=env)

        override_filenames = _generate_overrides(test_dir, filename2, name,
                                                 version)
        ctx = Context(filename2, override_filenames, id_, name, version,
                      revision)
        ctxs.append(ctx)

    sample_cmds = {}
    for language in languages:
        func = _LOAD_FUNCS[language]
        for k, v in func(test_dir, ctxs).iteritems():
            if k not in sample_cmds:
                sample_cmds[k] = {}
            sample_cmds[k][language] = v

    for ctx in ctxs:
        cmd = 'python {}.{}.mock.py'.format(ctx.name, ctx.version)
        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version)
        server = subprocess.Popen(shlex.split(cmd), cwd=src_dir, env=env,
                                  stderr=subprocess.PIPE)
        while not server.stderr.readline():
            pass
        time.sleep(1)

        bold = lambda x: '\033[95m{}\033[0m'.format(x)
        green = lambda x: '\033[92m{}\033[0m'.format(x)
        red = lambda x: '\033[91m{}\033[0m'.format(x)

        print('\n{}'.format(bold(ctx.id_)))
        for language in languages:
            err_logs = {}
            fail = False
            n = len(sample_cmds[ctx.id_][language])
            i = 0

            print('{0:<7}'.format(language), end='')
            for cmd in sample_cmds[ctx.id_][language]:
                sys.stdout.flush()

                sample = subprocess.Popen(shlex.split(cmd.command),
                                          cwd=cmd.cwd, env=env,
                                          stdout=subprocess.PIPE,
                                          stderr=subprocess.PIPE)
                # This call waits for the process to terminate.
                stdout_data, stderr_data = sample.communicate()

                i += 1
                print('.'*(int(i*10./n) - int((i-1)*10./n)), end='')

                # The sample fails if the return code is non-zero, or if the
                # language is Node.js and anything is written to stderr.
                cond = bool(sample.returncode)
                cond = cond or (language == _NODEJS and bool(stderr_data))
                # This is a safety check to make sure we don't miss false
                # positives in responses returned by the Node.js client
                # library. Specifically, the client may not error if the
                # response from the server is an HTML page. Instead, it will
                # print that HTML page to stdout as a JSON string.
                cond = cond or stdout_data.startswith('"<')
                if cond:
                    # Record the failure to the error log and mark failure.
                    err_logs[cmd.id_] = (stdout_data, stderr_data)
                    fail = True

            if fail:
                print(red(' FAIL'))
            else:
                print(green(' OK'))

            indent = lambda x: '\n'.join((4*' ') + y for y in x.splitlines())

            if err_logs:
                print('')
            for k, v in err_logs.items():
                log = '    {}\n'.format(k)
                if v[0]:
                    log += '    --- stdout\n'
                    log += indent(v[0])
                if v[1]:
                    log += '    --- stderr'
                    log += indent(v[1])
                print(red(log), end='\n\n')

        server.terminate()
        server.wait()


def _main():
    parser = argparse.ArgumentParser()
    langs = _LANGUAGES
    parser.add_argument('-l', action='append', choices=langs)
    parser.add_argument('files', metavar='FILE', nargs='+')
    args = parser.parse_args()

    # If any languages were specified, use those as the languages list instead.
    if args.l:
        langs = sorted(set(args.l))

    _run(args.files, langs)


if __name__ == '__main__':
    _main()
