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
import time

_GAPIC_YAML_FILENAMES = {
    'csharp': 'toolkit/src/main/resources/com/google/api/codegen/csharp/csharp_discovery.yaml',
    'go': 'toolkit/src/main/resources/com/google/api/codegen/go/go_discovery.yaml',
    'java': 'toolkit/src/main/resources/com/google/api/codegen/java/java_discovery.yaml',
    'nodejs': 'toolkit/src/main/resources/com/google/api/codegen/nodejs/nodejs_discovery.yaml',
    'php': 'toolkit/src/main/resources/com/google/api/codegen/php/php_discovery.yaml',
    'python': 'toolkit/src/main/resources/com/google/api/codegen/py/python_discovery.yaml',
    'ruby': 'toolkit/src/main/resources/com/google/api/codegen/ruby/ruby_discovery.yaml'
}

def _init_csharp_lib(ctx):
    client_lib_dir = os.path.join(ctx.lib_dir, 'google-api-dotnet-client')
    client_generator_dir = os.path.join(client_lib_dir, 'ClientGenerator')
    if not os.path.exists(client_lib_dir):
        cmd = ('git clone --depth 1'
               ' https://github.com/google/google-api-dotnet-client')
        subprocess.check_call(shlex.split(cmd), cwd=ctx.lib_dir)
        cmd = 'virtualenv venv'
        subprocess.check_call(shlex.split(cmd), cwd=client_generator_dir)
        cmd = 'venv/bin/python setup.py install'
        subprocess.check_call(shlex.split(cmd), cwd=client_generator_dir)

    cmd = ('venv/bin/python src/googleapis/codegen/generate_library.py'
           ' --input {}'
           ' --language csharp'
           ' --output_dir ../Src/Generated').format(ctx.discovery_doc_filename)
    subprocess.check_call(shlex.split(cmd), cwd=client_generator_dir)

def _init_csharp_env(ctx):
    client_lib_dir = os.path.join(ctx.lib_dir, 'google-api-dotnet-client')

    title = lambda x: x[0].upper() + x[1:] if x else x
    name = ctx.canonical_name
    if not name:
        name = ctx.name
    service_name = ''.join([title(x) for x in re.compile(r'[\._/-]+').split(name)])
    version_name = ctx.version.replace('.', '_').replace('-', '')
    service_dir = os.path.join(client_lib_dir,
            'Src/Generated/Google.Apis.{}.{}'.format(service_name, version_name))

    print service_dir
    cmd = 'dotnet migrate'
    subprocess.check_call(shlex.split(cmd), cwd=service_dir)

    csharp_src_dir = '{}/csharp'.format(ctx.src_dir)
    if not os.path.exists(csharp_src_dir):
        os.makedirs(csharp_src_dir)
    cmd = 'dotnet new sln -n app'
    subprocess.check_call(shlex.split(cmd), cwd=csharp_src_dir)

    csproj_filenames = []
    for filename in glob.glob('{}/*.frag.cs'.format(ctx.autogen_src_dir)):
        partname = os.path.split(filename)[1][:-len('.frag.cs')]
        frag_dir = '{}/{}'.format(csharp_src_dir, partname)
        if not os.path.exists(frag_dir):
            os.makedirs(frag_dir)
        shutil.copy2(filename, '{}/Program.cs'.format(frag_dir))
        csproj_filename = '{}/{}.csproj'.format(frag_dir, partname)
        csproj_filenames.append('{}/{}.csproj'.format(partname, partname))
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
""".format('{}/Src/Generated/Google.Apis.{}.{}/Google.Apis.{}.{}.csproj'.format(client_lib_dir, service_name, version_name, service_name, version_name)))
    cmd = 'dotnet sln app.sln add {}'.format(' '.join(csproj_filenames))
    subprocess.check_call(shlex.split(cmd), cwd=csharp_src_dir)
    cmd = 'dotnet restore'
    subprocess.check_call(shlex.split(cmd), cwd=csharp_src_dir)
    cmd = 'dotnet msbuild /m'
    subprocess.check_call(shlex.split(cmd), cwd=csharp_src_dir)

def _init_lang_lib(lang, ctx):
    if lang == 'csharp':
        _init_csharp_lib(ctx)
    else:
        raise Exception('unknown language: {}'.format(lang))

def _init_lang_env(lang, ctx):
    if lang == 'csharp':
        _init_csharp_env(ctx)
    else:
        raise Exception('unknown language: {}'.format(lang))

def _write_override_files(ctx):
    filenames = []
    name_dot_version = '{}.{}'.format(ctx.name, ctx.version)
    dv_override_filename = '{}.override1.json'.format(name_dot_version)
    dv_override_filename = os.path.join(ctx.src_dir, dv_override_filename)
    filenames.append(dv_override_filename)
    cmd = 'python gen_ovr.py {} --output {}'.format(
            ctx.discovery_doc_filename, dv_override_filename)
    subprocess.call(shlex.split(cmd))

    suffix = 2
    override_filename = os.path.splitext(ctx.discovery_doc_filename)[0]
    override_filename = '{}.override.json'.format(override_filename)
    if os.path.isfile(override_filename):
        orig_override_filename = '{}.override2.json'.format(name_dot_version)
        orig_override_filename = os.path.join(ctx.src_dir, 'override2.json')
        filenames.append(orig_override_filename)
        shutil.copy2(override_filename, orig_override_filename)
        suffix += 1

    auth_override_filename = '{}.override{}.json'.format(name_dot_version, suffix)
    auth_override_filename = os.path.join(ctx.src_dir, auth_override_filename)
    filenames.append(auth_override_filename)
    with open(auth_override_filename, 'w') as file_:
        discovery_doc_url = 'http://localhost:8000/static/{}.json'.format(
                name_dot_version)
        auth_override = {
            'csharp|go|java|js|nodejs|php|python|ruby': {
                'authType': 'NONE'
            },
            'js|python': {
                'discoveryDocUrl': discovery_doc_url
            }
        }
        file_.write(json.dumps(auth_override, sort_keys=True, indent=2))
    return filenames

def main():
    parser = argparse.ArgumentParser()
    langs = ['csharp', 'go', 'java', 'nodejs', 'php', 'python', 'ruby']
    parser.add_argument('-l', action='append', choices=langs)
    parser.add_argument('ids', metavar='ID', nargs='+')
    args = parser.parse_args()

    # If any languages were specified, use those as the languages list instead.
    if args.l:
        langs = sorted(set(args.l))

    # Note: We can't use the tempfile module because the directory it creates
    # is not accessible by package managers. For example, dotnet restore
    # doesn't work in a tempfile created directory.
    # We also can't use the /tmp directory for the same reason...
    temp_dir = os.path.abspath('test/{}'.format(int(time.time())))
    if not os.path.exists(temp_dir):
        os.makedirs(temp_dir)
    print('temp_dir: {}'.format(temp_dir))

    lib_dir = os.path.join(temp_dir, 'lib')
    if not os.path.exists(lib_dir):
        os.makedirs(lib_dir)

    Context = collections.namedtuple('Context',
            ('name canonical_name version revision discovery_doc_filename autogen_src_dir src_dir'
             ' lib_dir'))
    ctxs = []
    for id_ in args.ids:
        name_version = id_.split(':')
        name, version = name_version[0], name_version[1]

        discovery_doc_filename = 'discoveries/{}.{}.json'.format(name, version)
        discovery_doc_filename = os.path.abspath(discovery_doc_filename)

        if not os.path.isfile(discovery_doc_filename):
            raise Exception('could not find file: {}'.format(discovery_doc_filename))

        root = {}
        canonical_name, revision = '', ''
        with open(discovery_doc_filename) as file_:
            root = json.load(file_)
            name = root['name']
            canonical_name = root.get('canonicalName', '')
            version = root['version']
            revision = root['revision']

        # Make temporary directories for sample source and client libraries.
        autogen_src_dir = os.path.join(temp_dir, 'autogenerated', name, version, revision)
        src_dir = os.path.join(temp_dir, 'src', name, version, revision)
        if not os.path.exists(src_dir):
            os.makedirs(src_dir)

        with open('{}/{}.{}.json'.format(src_dir, name, version), 'w') as file_:
            root['rootUrl'] = 'http://localhost:8080/'
            file_.write(json.dumps(root, sort_keys=True, indent=2))

        ctxs.append(Context(name, canonical_name, version, revision,
                '{}/{}.{}.json'.format(src_dir, name, version),
                autogen_src_dir, src_dir, lib_dir))
    if not ctxs:
        raise Exception('no IDs to test')

    cmd = './gradlew discoJar'
    subprocess.check_call(shlex.split(cmd), cwd='toolkit')

    for ctx in ctxs:
        for lang in langs:
            _init_lang_lib(lang, ctx)

    for ctx in ctxs:
        override_filenames = _write_override_files(ctx)
        for lang in langs:
            cmd = ('java -jar toolkit/build/libs/discoGen-0.0.5.jar'
                   ' --discovery_doc {}'
                   ' --gapic_yaml {}'
                   ' --overrides {}'
                   ' --output {}').format(ctx.discovery_doc_filename,
                                          _GAPIC_YAML_FILENAMES[lang],
                                          ','.join(override_filenames),
                                          temp_dir)
            subprocess.check_call(shlex.split(cmd))

            _init_lang_env(lang, ctx)


if __name__ == '__main__':
    main()
