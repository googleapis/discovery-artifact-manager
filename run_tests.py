#!/usr/bin/env python3

from __future__ import absolute_import
from __future__ import print_function
import argparse
import collections
import concurrent.futures
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
import six

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
      <plugin>
        <groupId>org.apache.maven.plugins</groupId>
        <artifactId>maven-compiler-plugin</artifactId>
        <version>3.1</version>
        <configuration>
          <source>1.7</source>
          <target>1.7</target>
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
    """Returns the method ID of a sample from its filename.

    For example, given "foo.bar.get.frag.go", this function will return
    "foo.bar.get".

    Args:
        sample_filename (string): A sample filename.

    Returns:
        string: A method ID.
    """
    # "foo/foo.bar.get.frag.go" -> "foo.bar.get.frag.go"
    name = os.path.basename(sample_filename)
    # "foo.bar.get.frag.go" -> "foo.bar.get"
    return name.rsplit('.', 2)[0]


def _make_lib_dir(test_dir):
    """Creates and returns the path to lib.

    Args:
        test_dir (string): The parent directory.

    Returns:
        string: The path to the lib directory.
    """
    lib_dir = os.path.join(test_dir, 'lib')
    if not os.path.exists(lib_dir):
        os.makedirs(lib_dir)
    return lib_dir


def _call(cmd, **kwargs):
    """A wrapper over subprocess.call that splits cmd with shlex.split"""
    return subprocess.call(shlex.split(cmd), **kwargs)

def _make_lib_google_api_client_generator(test_dir):
    """Creates and returns the path to lib/google-api-client-generator

    The returned path points to a directory which contains source copied
    from google-api-client-generator and an initialized virtualenv in the
    "venv" subdirectory.

    Args:
        test_dir (string): The parent directory.

    Returns:
        string: The path to the lib/google-api-client-generator directory.
    """
    lib_dir = _make_lib_dir(test_dir)
    client_generator_dir = os.path.join(lib_dir, 'google-api-client-generator')
    if not os.path.exists(client_generator_dir):
        _call('cp -r google-api-client-generator {}'.format(lib_dir))
        _call('virtualenv venv', cwd=client_generator_dir)
        _call('venv/bin/python setup.py install', cwd=client_generator_dir)
    return client_generator_dir


def _make_lib_venv(test_dir):
    """Creates and returns the path to lib/venv.

    The returned path points to an initialized Python3 virtualenv.

    Args:
        test_dir (string): The parent directory.

    Returns:
        string: The path to the lib/venv directory.
    """
    venv_dir = os.path.join(_make_lib_dir(test_dir), 'venv')
    if not os.path.exists(venv_dir):
        _call('python3 -m venv {}'.format(venv_dir))

        env = os.environ.copy()
        env['PATH'] = '{}:{}'.format(os.path.join(venv_dir, 'bin'),
                                     env['PATH'])
        _call('python3 setup.py install', cwd='mockgen', env=env)
        _call('pip3 install flask gunicorn', env=env)
    return venv_dir


def _make_src_dir(test_dir, name, version, language=None):
    """Creates and returns the path to a src directory.

    For example, given ("/tmp", "foo", "v1"), this function will return:
        "/tmp/foo/v1"
    Given ("/tmp", "foo", "v1", "java"), this function will return:
        "/tmp/foo/v1/java"

    Args:
        test_dir (string): The parent directory.
        name (string): The API name.
        version (string): The API version.
        language (string): The language name.

    Returns:
        string: The path to a src directory.
    """
    src_dir = os.path.join(test_dir, 'src', name, version)
    if language:
        src_dir = os.path.join(src_dir, language)
    if not os.path.exists(src_dir):
        os.makedirs(src_dir)
    return src_dir


def _generate_overrides(test_dir, discovery_document_filename, name, version):
    """Returns an array of generated override filenames for the given API.

    For the given Discovery document, this method generates up to 3 override files:
    1. A mock value override which overrides strings with defined patterns.
    2. The original override file copied into the src directory.
    3. A mock auth/discoveryDocUrl override which forces the sample to be
       generated with no auth and for any Discovery document URLs to point to
       the mock server.

    Args:
        test_dir: The parent directory.
        discovery_document_filename: The Discovery document's filename.
        name: The API name.
        version: The API version.

    Returns:
        list: A list of filename strings.
    """
    src_dir = _make_src_dir(test_dir, name, version)
    filenames = []
    value_override_filename = os.path.join(src_dir, 'override1.json')
    filenames.append(value_override_filename)

    venv_dir = _make_lib_venv(test_dir)
    # Temporarily create a new env that points to venv/bin so we can use the
    # scripts installed by the mockgen module.
    env = os.environ.copy()
    env['PATH'] = '{}:{}'.format(os.path.join(venv_dir, 'bin'), env['PATH'])

    # 1. Generate the mock value override.
    _call('generate_mock_value_override {} --output {}'.format(
            discovery_document_filename, value_override_filename), env=env)

    # 2. Try to copy over the original override file if it exists.
    suffix = 2
    override_filename = '{}.override.json'.format(
            os.path.splitext(discovery_document_filename)[0])
    if os.path.isfile(override_filename):
        filename = os.path.join(src_dir, 'override2.json')
        filenames.append(filename)
        shutil.copy2(original_override_filename, filename)
        suffix += 1

    # 3. Generate the auth/discoveryDocUrl override.
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
    """Generates and returns the filenames of samples for an API/language.

    All samples are written to a temporary directory.

    Args:
        ctx (Context): The Context to generate from.
        language (string): The language to generate for.
        ruby_names_file (string, optional): The path to the Ruby names file.

    Returns:
        list: A list of generated sample filenames.
    """
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
    _call(cmd)
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
    """Loads the C# library and samples for the given APIs.

    Args:
        test_dir: The parent directory.
        ctxs: The list of Contexts to load.

    Returns:
        list: A list of Commands to run samples.
    """
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-dotnet-client')

    # Clone the client and reset it to v1.26.2 (latest).
    _call('git clone https://github.com/google/google-api-dotnet-client',
          cwd=lib_dir)
    _call('git reset --hard v1.26.2', cwd=client_lib_dir)

    # Delete and recreate the DiscoveryJson/ directory.
    discovery_json_dir = os.path.join(client_lib_dir, 'DiscoveryJson')
    shutil.rmtree(discovery_json_dir)
    os.makedirs(discovery_json_dir)

    # Copy all Discovery documents into DiscoveryJson/
    for ctx in ctxs:
        shutil.copy(ctx.discovery_document_filename, discovery_json_dir)

    # Create the virtualenv for google-api-dotnet-client's local copy of
    # google-api-client-generator.
    client_generator_dir = os.path.join(client_lib_dir, 'ClientGenerator')
    _call('virtualenv venv', cwd=client_generator_dir)
    _call('venv/bin/python setup.py install', cwd=client_generator_dir)

    # Temporarily create a new env that points to venv/bin so the bash script
    # can use the scripts installed by google-api-client-generator.
    venv_bin_dir = os.path.join(client_generator_dir, 'venv', 'bin')
    env = os.environ.copy()
    env['PATH'] = '{}:{}'.format(venv_bin_dir, env['PATH'])

    # Run the repo's generator (builds client libraries for all Discovery files
    # under DiscoveryJson/).
    _call('bash BuildGenerated.sh --onlygenerate', cwd=client_lib_dir, env=env)

    # Remove and recreate "Generated.sln".
    _call('rm {}'.format(os.path.join(client_lib_dir, 'Generated.sln')),
          cwd=client_lib_dir)
    csproj_filenames = glob.glob(os.path.join(client_lib_dir, 'Src',
                                              'Generated', '*', '*.csproj'))
    _call('dotnet new sln --name Generated', cwd=client_lib_dir)
    _call('dotnet sln Generated.sln add {}'.format(' '.join(csproj_filenames)),
          cwd=client_lib_dir)

    # Restore and build with framework netstandard1.3
    # Building here saves us considerable time when building the project
    # samples since we can ignore dependencies.
    _call('dotnet restore Generated.sln', cwd=client_lib_dir)
    _call('dotnet build --framework netstandard1.3 /m', cwd=client_lib_dir)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _CSHARP)
        sample_cmds[ctx.id_] = []

        # Create .../test.sln to collect all the sample projects.
        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _CSHARP)
        _call('dotnet new sln --name test', cwd=src_dir)

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

            dll_filename = os.path.join(project_dir, 'bin', 'Release',
                                        'netcoreapp1.0',
                                        '{}.dll'.format(method_id))
            # dotnet foo.bar.get/bin/Release/netcoreapp1.0/foo.bar.get.dll
            cmd = 'dotnet {}'.format(dll_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, None))

        # Add all projects to test.sln, restore, and build. Use multiple
        # processors if possible (/m flag).
        _call('dotnet sln test.sln add {}'.format(' '.join(csproj_filenames)),
              cwd=src_dir)
        _call('dotnet restore', cwd=src_dir)
        _call('dotnet build --no-dependencies --configuration Release /m',
              cwd=src_dir)

    return sample_cmds


def _load_go(test_dir, ctxs):
    """Loads the Go library and samples for the given APIs.

    Args:
        test_dir: The parent directory.
        ctxs: The list of Contexts to load.

    Returns:
        list: A list of Commands to run samples.
    """
    lib_dir = _make_lib_dir(test_dir)
    go_dir = os.path.join(lib_dir, 'go')
    if not os.path.exists(go_dir):
        os.makedirs(go_dir)
    # Temporarily create a new env that points GOPATH to lib/go
    env = os.environ.copy()
    env['GOPATH'] = go_dir
    _call('go get -v google.golang.org/api/google-api-go-generator', env=env)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _GO)
        sample_cmds[ctx.id_] = []

        # It's impossible to figure out where any Go API is without copying the
        # generator's logic for version names...
        version = ctx.version
        if version == 'alpha' or version == 'beta':
            version = 'v0.' + version
        match = re.match(r'^(.+)_(v[\d\.]+)$', version)
        if match:
            version = '{}/{}'.format(match.group(1), match.group(2))
        _call('{} --api_json_file {}'.format(
                os.path.join(go_dir, 'bin', 'google-api-go-generator'),
                ctx.discovery_document_filename), env=env)

        # Point GOBIN to ./bin so we can collect the executables without fear
        # of collisions under GOPATH/bin
        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _GO)
        bin_dir = os.path.join(src_dir, 'bin')
        env['GOBIN'] = bin_dir
        cmd = 'ln -s {} {}'
        _call(cmd.format(src_dir, os.path.join(go_dir, 'src', ctx.id_)))

        sample_cmds[ctx.id_] = []
        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            # ./foo.bar.get
            package_dir = os.path.join(src_dir, method_id)
            if not os.path.exists(package_dir):
                os.makedirs(package_dir)
            # ./foo.bar.get/foo.bar.get.go
            new_filename = os.path.join(package_dir, '{}.go'.format(method_id))
            shutil.copy(filename, new_filename)
            # ./bin/foo.bar.get
            cmd = os.path.join(bin_dir, method_id)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, None))

        # Compile all source and get all dependencies. This writes all
        # executables to ./bin
        _call('go get -v ./...', cwd=src_dir, env=env)

    return sample_cmds


def _load_java(test_dir, ctxs):
    """Loads the Java library and samples for the given APIs.

    Args:
        test_dir: The parent directory.
        ctxs: The list of Contexts to load.

    Returns:
        list: A list of Commands to run samples.
    """
    lib_dir = _make_lib_dir(test_dir)
    client_generator_dir = _make_lib_google_api_client_generator(test_dir)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _JAVA)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _JAVA)
        mvn_src_dir = os.path.join(src_dir, 'src', 'main', 'java')
        # Generate the client library and put it in src/main/java
        cmd = ('venv/bin/python src/googleapis/codegen/generate_library.py'
               ' --input {}'
               ' --language java'
               ' --language_variant 1.22.0'
               ' --package_path api/services'
               ' --output_dir {}')
        _call(cmd.format(ctx.discovery_document_filename, mvn_src_dir),
              cwd=client_generator_dir)
        with open(os.path.join(src_dir, 'pom.xml'), 'w') as file_:
            file_.write(_POM_XML)

        for i, filename in enumerate(sample_filenames):
            method_id = _parse_method_id_from_sample_filename(filename)
            package_dir = os.path.join(mvn_src_dir, method_id)
            if not os.path.exists(package_dir):
                os.makedirs(package_dir)

            # Match and replace the filename with the sample's class name.
            sample_class_name = ''
            sample_content = ''
            with open(filename) as file_:
                sample_content = file_.read()
                match = re.search(r'class\s+(\w+)', sample_content)
                sample_class_name = match.group(1)
            new_filename = os.path.join(package_dir,
                                        '{}.java'.format(sample_class_name))

            # Prepend "package main;" to each sample.
            with open(new_filename, 'w') as file_:
                file_.write('package p{};\n'.format(i))
                file_.write(sample_content)
            jar_filename = 'app-1.0-jar-with-dependencies.jar'
            jar_filename = os.path.join(src_dir, 'target', jar_filename)
            # java -cp .../target/app-bla.jar foo.bar.get.FooSample
            cmd = 'java -cp {} p{}.{}'.format(jar_filename, i,
                                              sample_class_name)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, None))

        # Assemble an executable jar.
        _call('mvn package assembly:single', cwd=src_dir)

    return sample_cmds


def _load_nodejs(test_dir, ctxs):
    """Loads the Node.js library and samples for the given APIs.

    Args:
        test_dir: The parent directory.
        ctxs: The list of Contexts to load.

    Returns:
        list: A list of Commands to run samples.
    """
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-nodejs-client')

    # Clone the client, install dependencies, build the scripts, and delete the
    # generated apis directory.
    _call(('git clone --depth 1 '
           'https://github.com/google/google-api-nodejs-client'), cwd=lib_dir)
    _call('npm install', cwd=client_lib_dir)
    _call('npm run build-tools', cwd=client_lib_dir)
    _call('rm -rf apis', cwd=client_lib_dir)

    # Generate all client libraries.
    for ctx in ctxs:
        cmd = 'node scripts/generate {}'
        _call(cmd.format(ctx.discovery_document_filename), cwd=client_lib_dir)

    # Build all client libraries.
    _call('npm run build', cwd=client_lib_dir)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _NODEJS)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _NODEJS)
        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            new_filename = '{}.js'.format(method_id)
            shutil.copy(filename, os.path.join(src_dir, new_filename))
            # node .../foo.bar.get.js
            cmd = 'node {}'.format(new_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, src_dir))

        _call('npm install {}'.format(client_lib_dir), cwd=src_dir)

    return sample_cmds


def _load_php(test_dir, ctxs):
    """Loads the PHP library and samples for the given APIs.

    Args:
        test_dir: The parent directory.
        ctxs: The list of Contexts to load.

    Returns:
        list: A list of Commands to run samples.
    """
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-php-client-services')
    _call(('git clone --depth 1'
           ' https://github.com/google/google-api-php-client-services'),
          cwd=lib_dir)
    client_generator_dir = _make_lib_google_api_client_generator(test_dir)

    # Delete all generated client libraries.
    shutil.rmtree(os.path.join(client_lib_dir, 'src'))

    # Generate all client libraries.
    for ctx in ctxs:
        cmd = ('venv/bin/python src/googleapis/codegen/generate_library.py'
               ' --input {}'
               ' --language php'
               ' --language_variant 1.2.0'
               ' --output_dir {}')
        _call(cmd.format(
                ctx.discovery_document_filename,
                os.path.join(client_lib_dir, 'src', 'Google', 'Service')),
              cwd=client_generator_dir)

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
            # php .../foo.bar.get.php
            cmd = 'php {}'.format(new_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, src_dir))

        # Install all dependencies.
        _call('composer update', cwd=src_dir)

    return sample_cmds


def _load_python(test_dir, ctxs):
    """Loads the Python library and samples for the given APIs.

    Args:
        test_dir: The parent directory.
        ctxs: The list of Contexts to load.

    Returns:
        list: A list of Commands to run samples.
    """
    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _PYTHON)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _PYTHON)
        # Create a virtualenv.
        _call('virtualenv venv', cwd=src_dir)
        _call('venv/bin/pip install google-api-python-client', cwd=src_dir)

        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            new_filename = '{}.py'.format(method_id)
            shutil.copy(filename, os.path.join(src_dir, new_filename))
            # /venv/bin/python .../foo.bar.get.py
            cmd = 'venv/bin/python {}'.format(new_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, src_dir))

    return sample_cmds


def _load_ruby(test_dir, ctxs):
    """Loads the Ruby library and samples for the given APIs.

    Args:
        test_dir: The parent directory.
        ctxs: The list of Contexts to load.

    Returns:
        list: A list of Commands to run samples.
    """
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-ruby-client')

    # Clone the client library, install dependencies, delete all generated
    # client libraries, and restore the discovery_v1 client (needed by the
    # generator script).
    _call(('git clone --depth 1'
           ' https://github.com/google/google-api-ruby-client'), cwd=lib_dir)
    _call('bundle install --path vendor/bundle', cwd=client_lib_dir)
    shutil.rmtree(os.path.join(client_lib_dir, 'generated'))
    _call('git checkout generated/google/apis/discovery_v1',
          cwd=client_lib_dir)
    _call('git checkout generated/google/apis/discovery_v1.rb',
          cwd=client_lib_dir)

    discovery_document_filenames = []
    for ctx in ctxs:
        discovery_document_filenames.append(ctx.discovery_document_filename)
    names_filename = os.path.join(client_lib_dir, 'api_names_out.yaml')

    # Generate all client libraries.
    cmd = ('bundle exec bin/generate-api gen generated --file {}'
           ' --names_out {}')
    cmd = cmd.format(' '.join(discovery_document_filenames), names_filename)
    proc = subprocess.Popen(shlex.split(cmd), cwd=client_lib_dir,
                            stdin=subprocess.PIPE)
    # The generate-api script asks for user input... 'a' means accept all.
    proc.communicate(input=b'a')
    proc.wait()

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _RUBY,
                                             ruby_names_file=names_filename)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _RUBY)

        # Create a Gemfile that points to lib/google-api-ruby-client
        with open(os.path.join(src_dir, 'Gemfile'), 'w') as file_:
            file_.write('source \'https://rubygems.org\'\n')
            line = 'gem \'google-api-client\', :path => \'{}\'\n'
            line = line.format(client_lib_dir)
            file_.write(line)

        for filename in sample_filenames:
            method_id = _parse_method_id_from_sample_filename(filename)
            new_filename = '{}.rb'.format(method_id)
            shutil.copy(filename, os.path.join(src_dir, new_filename))
            # bundle exec ruby .../foo.bar.get.rb
            cmd = 'bundle exec ruby {}'.format(new_filename)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, src_dir))

        # Install all dependencies.
        _call('bundle install --path vendor/bundle', cwd=src_dir)

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


def _work(cmd):
    """Returns the result of the process run from Command.

    A worker function to parallelize the sample tests.

    Args:
        cmd (Command): The command to run.

    Returns:
        tuple: A tuple containing the return code of the process and the output
        of STDIN and STDOUT: (returncode, (stdin_out, stdout_out)).
    """
    proc = subprocess.Popen(shlex.split(cmd.command), cwd=cmd.cwd,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    proc.wait()
    return (proc.returncode, proc.communicate())


def _run(discovery_document_filenames, languages):
    test_dir = os.path.abspath('test/{}'.format(int(time.time())))
    if not os.path.exists(test_dir):
        os.makedirs(test_dir)

    roots = []
    for filename in discovery_document_filenames:
        with open(filename) as file_:
            roots.append(json.load(file_))

    venv_dir = (_make_lib_venv(test_dir))
    # Temporarily create a new env that points to venv/bin so we can use the
    # scripts installed by the mockgen module.
    env = os.environ.copy()
    env['PATH'] = '{}:{}'.format(os.path.join(venv_dir, 'bin'), env['PATH'])

    # Generate a Context for each passed Discovery document.
    ctxs = []
    for root in roots:
        id_ = root['id']
        name = root['name']
        version = root['version']
        revision = root['revision']

        src_dir = _make_src_dir(test_dir, name, version)

        filename2 = '{}.{}.json'.format(name, version)
        filename2 = os.path.join(src_dir, filename2)
        _call('generate_mock_discovery_document {} --output {}'.format(
                filename, filename2), env=env)
        _call('generate_mock_server {} --directory {}'.format(
                filename2, src_dir), env=env)

        override_filenames = _generate_overrides(test_dir, filename2, name,
                                                 version)
        ctx = Context(filename2, override_filenames, id_, name, version,
                      revision)
        ctxs.append(ctx)

    # Load all Contexts for all languages.
    sample_cmds = {}
    for language in languages:
        func = _LOAD_FUNCS[language]
        for k, v in six.iteritems(func(test_dir, ctxs)):
            if k not in sample_cmds:
                sample_cmds[k] = {}
            sample_cmds[k][language] = v

    for ctx in ctxs:
        # Run the server with gunicorn so it doesn't lock up when responding to
        # requests made by multiple threads/processes.
        cmd = 'gunicorn -w 4 server:app'.format(ctx.name, ctx.version)
        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version)
        time.sleep(4)
        server = subprocess.Popen(shlex.split(cmd), cwd=src_dir, env=env,
                                  stderr=subprocess.PIPE)
        while not server.stderr.readline():
            pass
        time.sleep(0.25)

        bold = lambda x: '\033[95m{}\033[0m'.format(x)
        green = lambda x: '\033[92m{}\033[0m'.format(x)
        red = lambda x: '\033[91m{}\033[0m'.format(x)

        # For each language, run each sample.
        print('\n' + bold(ctx.id_))
        for language in languages:
            err_logs = {}
            fail = False
            n = len(sample_cmds[ctx.id_][language])
            i = 0

            print('{0:<7}'.format(language), end='')

            with concurrent.futures.ProcessPoolExecutor() as ex:
                cmds = sample_cmds[ctx.id_][language]
                method_ids = [x.id_ for x in cmds]
                for method_id, result in zip(method_ids, ex.map(_work, cmds)):
                    sys.stdout.flush()
                    stdout_data, stderr_data = result[1]

                    i += 1
                    print('.'*(int(i*10./n) - int((i-1)*10./n)), end='')

                    # The sample fails if returncode != 0, or if the language
                    # is Node.js and anything is written to stderr.
                    cond = bool(result[0])
                    cond = cond or (language == _NODEJS and bool(stderr_data))
                    # This is a safety check to make sure we don't miss false
                    # positives in responses returned by the Node.js client
                    # library. Specifically, the client may not error if the
                    # response from the server is an HTML page. Instead, it
                    # will print that HTML page to stdout as a JSON string.
                    cond = cond or stdout_data.startswith(b'"<')
                    if cond:
                        # Record the failure to the error log and mark failure.
                        err_logs[method_id] = (stdout_data, stderr_data)
                        fail = True

            if fail:
                print(red(' FAIL'))
            else:
                print(green(' OK'))

            # Simple lambda to indent the input string by 4 spaces.
            indent = lambda x: '\n'.join((4*' ') + y for y in x.splitlines())

            if err_logs:
                print('')
            for k in sorted(err_logs):
                log = '    {}\n'.format(k)
                v = err_logs[k]
                v = [indent(x.decode('utf-8').strip('\n')) for x in v]
                if v[0]:
                    log += '\n    --- stdout\n'
                    log += indent(v[0]) + '\n'
                if v[1]:
                    log += '\n    --- stderr\n'
                    log += indent(v[1]) + '\n'
                print(red(log), end='\n')

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
