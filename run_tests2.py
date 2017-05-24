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
import tempfile
import time

_DEVNULL = open(os.devnull, 'w')

_CSHARP = 'csharp'
_GO = 'go'
_JAVA = 'java'

_LANGUAGES = [_CSHARP, _GO, _JAVA, 'nodejs', 'php', 'python', 'ruby']

_GAPIC_YAML_FILENAMES = {
    _CSHARP: ('toolkit/src/main/resources/com/google/api/codegen/csharp'
               '/csharp_discovery.yaml'),
    _GO: ('toolkit/src/main/resources/com/google/api/codegen/go'
           '/go_discovery.yaml'),
    _JAVA: ('toolkit/src/main/resources/com/google/api/codegen/java'
             '/java_discovery.yaml'),
    'nodejs': ('toolkit/src/main/resources/com/google/api/codegen/nodejs'
               '/nodejs_discovery.yaml'),
    'php': ('toolkit/src/main/resources/com/google/api/codegen/php'
            '/php_discovery.yaml'),
    'python': ('toolkit/src/main/resources/com/google/api/codegen/py'
               '/python_discovery.yaml'),
    'ruby': ('toolkit/src/main/resources/com/google/api/codegen/ruby'
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


def _make_src_dir(test_dir, name, version, language=None):
    src_dir = os.path.join(test_dir, 'src', name, version)
    if language:
        src_dir = os.path.join(src_dir, language)
    if not os.path.exists(src_dir):
        os.makedirs(src_dir)
    return src_dir


def _generate_overrides(src_dir, discovery_document_filename, name, version):
    # TODO: Want to set up a virtualenv in test probably.
    filenames = []
    value_override_filename = os.path.join(src_dir, 'override1.json')
    filenames.append(value_override_filename)
    cmd = 'python mockgen/generate_mock_value_override.py {} --output {}'
    cmd = cmd.format(discovery_document_filename,
                     value_override_filename)
    subprocess.call(shlex.split(cmd))

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


def _generate_samples(ctx, language):
    temp_dir = tempfile.mkdtemp()
    cmd = ('java -jar discoGen-0.0.5.jar'
           ' --discovery_doc {} '
           ' --gapic_yaml {} '
           ' --overrides {}'
           ' --output {}').format(ctx.discovery_document_filename,
                                  _GAPIC_YAML_FILENAMES[language],
                                  ','.join(ctx.override_filenames),
                                  temp_dir)
    subprocess.call(shlex.split(cmd))
    autogen_src_dir = os.path.join(temp_dir, 'autogenerated', ctx.name,
                                   ctx.version, ctx.revision)
    ext = {
        _CSHARP: 'cs', _GO: 'go', _JAVA: 'java', 'nodejs': 'njs',
        'php': 'php', 'python': 'py', 'ruby': 'rb'
    }[language]
    sample_filenames = glob.glob(os.path.join(autogen_src_dir,
                                              '*.{}'.format(ext)))
    return sample_filenames


def _load_csharp(test_dir, ctxs):
    lib_dir = _make_lib_dir(test_dir)
    client_lib_dir = os.path.join(lib_dir, 'google-api-dotnet-client')

    cmd = ('git clone https://github.com/google/google-api-dotnet-client')
    subprocess.call(shlex.split(cmd), cwd=lib_dir)
    cmd = 'git reset --hard 65c178a116fa29c9945c9f9c6b8cfd457706f1ef'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)

    client_generator_dir = os.path.join(client_lib_dir, 'ClientGenerator')
    cmd = 'virtualenv venv'
    subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
    cmd = 'venv/bin/python setup.py install'
    subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
    cmd = 'dotnet migrate Src/Support'
    subprocess.call(shlex.split(cmd), cwd=client_lib_dir)

    generated_dir = os.path.join(client_lib_dir, 'Src', 'Generated')
    cmd = 'rm -rf {}'.format(generated_dir)
    subprocess.check_call(shlex.split(cmd))

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _CSHARP)
        sample_cmds[ctx.id_] = []

        cmd = ('venv/bin/python src/googleapis/codegen/generate_library.py'
               ' --input {}'
               ' --language csharp'
               ' --output_dir ../Src/Generated')
        cmd = cmd.format(ctx.discovery_document_filename)
        subprocess.call(shlex.split(cmd), cwd=client_generator_dir)

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

    cmd = 'dotnet migrate -s .'
    subprocess.call(shlex.split(cmd), cwd=generated_dir)
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
    env = os.environ
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
            shutil.copy(filename, os.path.join(package_dir,
                                               '{}.go'.format(method_id)))
            cmd = os.path.join(bin_dir, method_id)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, None))

        cmd = 'go get -v ./...'
        subprocess.call(shlex.split(cmd), cwd=src_dir, env=env)

    return sample_cmds


def _load_java(test_dir, ctxs):
    lib_dir = _make_lib_dir(test_dir)
    cmd = 'cp -r google-api-client-generator {}'.format(lib_dir)
    subprocess.call(shlex.split(cmd))

    client_generator_dir = os.path.join(lib_dir, 'google-api-client-generator')
    cmd = 'virtualenv venv'
    subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
    cmd = 'venv/bin/python setup.py install'
    subprocess.call(shlex.split(cmd), cwd=client_generator_dir)

    sample_cmds = {}
    for ctx in ctxs:
        sample_filenames = _generate_samples(ctx, _JAVA)
        sample_cmds[ctx.id_] = []

        src_dir = _make_src_dir(test_dir, ctx.name, ctx.version, _CSHARP)
        mvn_src_dir = os.path.join(src_dir, 'src', 'main', 'java')
        cmd = ('venv/bin/python src/googleapis/codegen/generate_library.py'
               ' --input {}'
               ' --language java'
               ' --package_path api/services'
               ' --output_dir {}')
        cmd = cmd.format(ctx.discovery_document_filename, mvn_src_dir)
        subprocess.call(shlex.split(cmd), cwd=client_generator_dir)
        with open(os.path.join(mvn_src_dir, 'pom.xml'), 'w') as file_:
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
            sample_class_filename = os.path.join(
                    package_dir, '{}.java'.format(sample_class_name))
            with open(sample_class_filename) as file_:
                file_.write('package {};\n'.format(method_id))
                file_.write(sample_content)

            jar_filename = 'app-1.0-jar-with-dependencies.jar'
            jar_filename = os.path.join(src_dir, 'target', jar_filename)
            cmd = 'java -cp {} {m}.{m}'.format(jar_filename, m=method_id)
            sample_cmds[ctx.id_].append(SampleCommand(method_id, cmd, None))

        cmd = 'mvn package assembly:single'
        subprocess.call(shlex.split(cmd), cwd=src_dir)

    return sample_cmds


_LOAD_FUNCS = {
    _CSHARP: _load_csharp,
    _GO: _load_go,
    _JAVA: _load_java
}


def _run(discovery_document_filenames, languages):
    test_dir = os.path.abspath('test/{}'.format(int(time.time())))
    if not os.path.exists(test_dir):
        os.makedirs(test_dir)

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
        cmd = ('python mockgen/generate_mock_discovery_document.py {}'
               ' --output {}').format(filename, filename2)
        subprocess.call(shlex.split(cmd))

        cmd = 'python mockgen/generate_mock_server.py {} --directory {}'
        cmd = cmd.format(filename2, src_dir)
        subprocess.call(shlex.split(cmd))

        override_filenames = _generate_overrides(src_dir, filename2, name,
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
        server = subprocess.Popen(shlex.split(cmd), cwd=src_dir,
                                  stderr=subprocess.PIPE)
        while not server.stderr.readline():
            pass
        time.sleep(0.1)

        print('\n\033[95m{}\033[0m'.format(ctx.id_))
        for language in languages:
            print('\033[95m{}\033[0m'.format(language))
            for cmd in sample_cmds[ctx.id_][language]:
                print('  {0:<48.48}'.format(cmd.id_), end='')
                sample = subprocess.Popen(shlex.split(cmd.command),
                                          cwd=cmd.cwd, stdout=_DEVNULL,
                                          stderr=subprocess.PIPE)
                sample.wait()
                if not sample.returncode:
                    print(' \033[92mOK\033[0m')
                else:
                    print(' \033[91mFAIL\033[0m')
                    print('  stderr:')
                    for line in proc.stderr.readline():
                        print('    {}'.format(line))

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
