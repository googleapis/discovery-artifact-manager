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
import time

# Create a class
# Store directory names as global variables

_DEVNULL = open(os.devnull, 'w')

_GAPIC_YAML_FILENAMES = {
    'csharp': 'toolkit/src/main/resources/com/google/api/codegen/csharp/csharp_discovery.yaml',
    'go': 'toolkit/src/main/resources/com/google/api/codegen/go/go_discovery.yaml',
    'java': 'toolkit/src/main/resources/com/google/api/codegen/java/java_discovery.yaml',
    'nodejs': 'toolkit/src/main/resources/com/google/api/codegen/nodejs/nodejs_discovery.yaml',
    'php': 'toolkit/src/main/resources/com/google/api/codegen/php/php_discovery.yaml',
    'python': 'toolkit/src/main/resources/com/google/api/codegen/py/python_discovery.yaml',
    'ruby': 'toolkit/src/main/resources/com/google/api/codegen/ruby/ruby_discovery.yaml'
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

def _init_csharp_lib(ctx):
    client_lib_dir = os.path.join(ctx.lib_dir, 'google-api-dotnet-client')
    client_generator_dir = os.path.join(client_lib_dir, 'ClientGenerator')
    if not os.path.exists(client_lib_dir):
        cmd = ('git clone --depth 1'
               ' https://github.com/google/google-api-dotnet-client')
        subprocess.check_call(shlex.split(cmd), cwd=ctx.lib_dir)
        #cmd = 'virtualenv venv'
        #subprocess.check_call(shlex.split(cmd), cwd=client_generator_dir)
        cmd = 'python setup.py install'
        subprocess.check_call(shlex.split(cmd), cwd=client_generator_dir)
        #cmd = 'dotnet migrate Src/Support'
        #subprocess.check_call(shlex.split(cmd), cwd=client_lib_dir)

    cmd = ('python src/googleapis/codegen/generate_library.py'
           ' --input {}'
           ' --language csharp'
           ' --output_dir ../Src/Generated').format(ctx.discovery_doc_filename)
    subprocess.check_call(shlex.split(cmd), cwd=client_generator_dir)


def _init_go_lib(ctx):
    go_dir = os.path.join(ctx.lib_dir, 'go')
    if not os.path.exists(go_dir):
        os.makedirs(go_dir)
    renamed_version = ctx.version
    odd_version_prog = re.compile('^(.+)_(v[\d\.]+)$')
    if ctx.version == 'alpha' or ctx.version == 'beta':
        renamed_version = 'v0.' + ctx.version
    match = odd_version_prog.match(ctx.version)
    if match:
        renamed_version = match.group(1) + '/' + match.group(2)
    env = os.environ
    env['GOPATH'] = go_dir
    cmd = 'go get google.golang.org/api/google-api-go-generator golang.org/x/net/context'
    print(cmd)
    subprocess.call(shlex.split(cmd), env=env)
    cmd = '{}/bin/google-api-go-generator --api_json_file {}'.format(go_dir, ctx.discovery_doc_filename)
    print(cmd)
    subprocess.call(shlex.split(cmd), env=env)


def _init_nodejs_lib(ctx):
    nodejs_dir = os.path.join(ctx.lib_dir, 'nodejs')
    client_dir = os.path.join(nodejs_dir, 'google-api-nodejs-client')
    if not os.path.exists(nodejs_dir):
        os.makedirs(nodejs_dir)
        cmd = 'git clone --depth 1 https://github.com/google/google-api-nodejs-client'
        subprocess.call(shlex.split(cmd), cwd=nodejs_dir)
        cmd = 'npm install'
        subprocess.call(shlex.split(cmd), cwd=client_dir)
        cmd = 'npm run build-tools'
        subprocess.call(shlex.split(cmd), cwd=client_dir)
    cmd = 'node scripts/generate {}'.format(ctx.discovery_doc_filename)
    subprocess.call(shlex.split(cmd), cwd=client_dir)


def _init_php_lib(ctx):
    php_dir = os.path.join(ctx.lib_dir, 'php')
    if not os.path.exists(php_dir):
        os.makedirs(php_dir)
        cmd = 'git clone --depth 1 https://github.com/google/google-api-php-client-services'
        subprocess.call(shlex.split(cmd), cwd=php_dir)
    if not os.path.exists('google-api-client-generator/venv'):
        cmd = 'virtualenv google-api-client-generator/venv'
        subprocess.call(shlex.split(cmd))
        cmd = 'venv/bin/python setup.py install'
        subprocess.call(shlex.split(cmd), cwd='google-api-client-generator')
    cmd = 'venv/bin/python src/googleapis/codegen/generate_library.py --input {} --language php --language_variant 1.2.0 --output_dir {}/google-api-php-client-services/src/Google/Service'.format(ctx.discovery_doc_filename, php_dir)
    subprocess.call(shlex.split(cmd), cwd='google-api-client-generator')


def _init_ruby_lib(ctx):
    ruby_dir = os.path.join(ctx.lib_dir, 'ruby')
    client_dir = os.path.join(ruby_dir, 'google-api-ruby-client')
    if not os.path.exists(ruby_dir):
        os.makedirs(ruby_dir)
        cmd = 'git clone --depth 1 https://github.com/google/google-api-ruby-client'
        subprocess.call(shlex.split(cmd), cwd=ruby_dir)
        cmd = 'bundle install --path vendor/bundle'
        subprocess.call(shlex.split(cmd), cwd=client_dir)
    cmd = 'bundle exec bin/generate-api gen generated --file {}'.format(ctx.discovery_doc_filename)
    ps = subprocess.Popen(['echo', 'a'], stdout=subprocess.PIPE)
    subprocess.call(shlex.split(cmd), cwd=client_dir, stdin=ps.stdout)
    print(cmd)


def _init_csharp_env(ctx):
    client_lib_dir = os.path.join(ctx.lib_dir, 'google-api-dotnet-client')
    for filename in glob.glob(os.path.join(client_lib_dir, 'DiscoveryJson', '*')):
        os.remove(filename)
    shutil.copy2(ctx.discovery_doc_filename, os.path.join(client_lib_dir, 'DiscoveryJson'))
    cmd = 'bash BuildGenerated.sh --skipdownload'
    subprocess.check_call(shlex.split(cmd), cwd=client_lib_dir)

    title = lambda x: x[0].upper() + x[1:] if x else x
    name = ctx.canonical_name.replace(' ', '')
    if not name:
        name = ctx.name
    service_name = ''.join([title(x) for x in re.compile(r'[\._/-]+').split(name)])
    version_name = ctx.version.replace('.', '_').replace('-', '')
    service_dir = os.path.join(client_lib_dir,
            'Src/Generated/Google.Apis.{}.{}'.format(service_name, version_name))

    #cmd = 'dotnet migrate'
    #subprocess.check_call(shlex.split(cmd), cwd=service_dir)

    csharp_src_dir = '{}/csharp'.format(ctx.src_dir)
    if not os.path.exists(csharp_src_dir):
        os.makedirs(csharp_src_dir)
    cmd = 'dotnet new sln -n app'
    subprocess.call(shlex.split(cmd), cwd=csharp_src_dir)

    cmds = []
    csproj_filenames = []
    for filename in glob.glob('{}/*.frag.cs'.format(ctx.autogen_src_dir)):
        partname = os.path.split(filename)[1][:-len('.frag.cs')]
        frag_dir = '{}/{}'.format(csharp_src_dir, partname)
        if not os.path.exists(frag_dir):
            os.makedirs(frag_dir)
        shutil.copy2(filename, '{}/Program.cs'.format(frag_dir))
        csproj_filename = '{}/{}.csproj'.format(frag_dir, partname)
        csproj_filenames.append('{}/{}.csproj'.format(partname, partname))
        piece = '{}/Src/Generated/Google.Apis.{}.{}/Google.Apis.{}.{}'.format(client_lib_dir, service_name, version_name, service_name, version_name)
        with open(csproj_filename, 'w') as file_:
            file_.write("""<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <ProjectReference Include="{}.csproj" />
  </ItemGroup>
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>netcoreapp1.0</TargetFramework>
  </PropertyGroup>
</Project>
""".format(piece))
        cmds.append(('dotnet {}/bin/Debug/netcoreapp1.0/{}.dll'.format(partname, partname), csharp_src_dir, partname))

    cmd = 'dotnet sln app.sln add {}'.format(' '.join(csproj_filenames))
    subprocess.check_call(shlex.split(cmd), cwd=csharp_src_dir)
    cmd = 'dotnet restore'
    subprocess.check_call(shlex.split(cmd), cwd=csharp_src_dir)
    cmd = 'dotnet msbuild /m'
    subprocess.check_call(shlex.split(cmd), cwd=csharp_src_dir)

    return cmds


def _init_go_env(ctx):
    go_src_dir = os.path.join(ctx.src_dir, 'go')
    if not os.path.exists(go_src_dir):
        os.makedirs(go_src_dir)
    go_bin_dir = os.path.join(go_src_dir, 'bin')
    env = os.environ
    env['GOBIN'] = go_bin_dir
    go_path = os.path.join(ctx.lib_dir, 'go')
    cmd = 'ln -s {} {}'.format(go_src_dir, os.path.join(go_path, '{}:{}'.format(ctx.name, ctx.version)))
    print(cmd)
    subprocess.call(shlex.split(cmd))

    cmds = []
    for filename in glob.glob('{}/*.frag.go'.format(ctx.autogen_src_dir)):
        partname = os.path.split(filename)[1][:-len('.frag.go')]
        cdir = os.path.join(go_src_dir, partname)
        os.makedirs(cdir)
        shutil.copy2(filename, '{}/{}.go'.format(cdir, partname))
        cmds.append(('bin/{}'.format(partname), go_src_dir, partname))
    cmd = 'go install -v ./...'
    subprocess.call(shlex.split(cmd), cwd=go_src_dir, env=env)
    return cmds


def _init_java_env(ctx):
    java_src_dir = os.path.join(ctx.src_dir, 'java')
    if not os.path.exists(java_src_dir):
        os.makedirs(java_src_dir)
    if not os.path.exists('google-api-client-generator/venv'):
        cmd = 'virtualenv google-api-client-generator/venv'
        subprocess.call(shlex.split(cmd))
        cmd = 'venv/bin/python setup.py install'
        subprocess.call(shlex.split(cmd), cwd='google-api-client-generator')
    cmd = 'venv/bin/python src/googleapis/codegen/generate_library.py --input {} --language java --package_path api/services --output_dir {}/src/main/java'.format(ctx.discovery_doc_filename, java_src_dir)
    subprocess.call(shlex.split(cmd), cwd='google-api-client-generator')
    with open(os.path.join(java_src_dir, 'pom.xml'), 'w') as file_:
        file_.write(_POM_XML)

    cmds = []
    for filename in glob.glob('{}/*.frag.java'.format(ctx.autogen_src_dir)):
        partname = os.path.split(filename)[1][:-len('.frag.java')]
        cdir = os.path.join(java_src_dir, 'src', 'main', 'java', partname)
        os.makedirs(cdir)
        class_name = ''
        data = ''
        with open(filename) as file_:
            data = file_.read()
            match = re.search(r'class\s+(\w+)', data)
            class_name = match.group(1)
        with open('{}/{}.java'.format(cdir, class_name), 'w') as file_:
            file_.write('package {};\n'.format(partname))
            file_.write(data)
        cmds.append(('java -cp target/app-1.0-jar-with-dependencies.jar {}.{}'.format(partname, class_name), java_src_dir, partname))
    cmd = 'mvn package assembly:single'
    subprocess.call(shlex.split(cmd), cwd=java_src_dir)
    return cmds


def _init_nodejs_env(ctx):
    nodejs_src_dir = os.path.join(ctx.src_dir, 'nodejs')
    if not os.path.exists(nodejs_src_dir):
        os.makedirs(nodejs_src_dir)
    client_dir = os.path.join(ctx.lib_dir, 'nodejs', 'google-api-nodejs-client')
    cmd = 'npm run build'
    subprocess.call(shlex.split(cmd), cwd=client_dir)
    cmd = 'npm install {}'.format(client_dir)
    subprocess.call(shlex.split(cmd), cwd=nodejs_src_dir)

    cmds = []
    for filename in glob.glob('{}/*.frag.njs'.format(ctx.autogen_src_dir)):
        partname = os.path.split(filename)[1][:-len('.frag.njs')]
        shutil.copy2(filename, '{}/{}.js'.format(nodejs_src_dir, partname))
        cmds.append(('node {}.js'.format(partname), nodejs_src_dir, partname))
    return cmds


def _init_php_env(ctx):
    php_src_dir = os.path.join(ctx.src_dir, 'php')
    if not os.path.exists(php_src_dir):
        os.makedirs(php_src_dir)
    client_dir = os.path.join(ctx.lib_dir, 'php', 'google-api-php-client-services')
    with open(os.path.join(php_src_dir, 'composer.json'), 'w') as file_:
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
""".format(client_dir))
    cmd = 'composer update'
    subprocess.call(shlex.split(cmd), cwd=php_src_dir)

    cmds = []
    for filename in glob.glob('{}/*.frag.php'.format(ctx.autogen_src_dir)):
        partname = os.path.split(filename)[1][:-len('.frag.php')]
        shutil.copy2(filename, '{}/{}.php'.format(php_src_dir, partname))
        cmds.append(('php {}.php'.format(partname), php_src_dir, partname))
    return cmds


def _init_python_env(ctx):
    python_src_dir = os.path.join(ctx.src_dir, 'python')
    if not os.path.exists(python_src_dir):
        os.makedirs(python_src_dir)
    if not os.path.exists(os.path.join(python_src_dir, 'venv')):
        cmd = 'virtualenv venv'
        subprocess.call(shlex.split(cmd), cwd=python_src_dir)
        cmd = 'venv/bin/pip install google-api-python-client'
        subprocess.call(shlex.split(cmd), cwd=python_src_dir)

    cmds = []
    for filename in glob.glob('{}/*.frag.py'.format(ctx.autogen_src_dir)):
        partname = os.path.split(filename)[1][:-len('.frag.py')]
        shutil.copy2(filename, '{}/{}.py'.format(python_src_dir, partname))
        cmds.append(('venv/bin/python {}.py'.format(partname), python_src_dir, partname))
    return cmds


def _init_ruby_env(ctx):
    ruby_src_dir = os.path.join(ctx.src_dir, 'ruby')
    if not os.path.exists(ruby_src_dir):
        os.makedirs(ruby_src_dir)
    with open('{}/Gemfile'.format(ruby_src_dir), 'w') as f:
        f.write('source \'https://rubygems.org\'\n')
        f.write('gem \'google-api-client\', :path => \'{}\'\n'.format(os.path.join(ctx.lib_dir, 'ruby', 'google-api-ruby-client')))
    cmd = 'bundle install --path vendor/bundle'
    subprocess.call(shlex.split(cmd), cwd=ruby_src_dir)

    cmds = []
    for filename in glob.glob('{}/*.frag.rb'.format(ctx.autogen_src_dir)):
        partname = os.path.split(filename)[1][:-len('.frag.rb')]
        shutil.copy2(filename, '{}/{}.rb'.format(ruby_src_dir, partname))
        cmds.append(('bundle exec ruby {}.rb'.format(partname), ruby_src_dir, partname))
    return cmds


def _init_lang_lib(lang, ctx):
    if lang == 'csharp':
        _init_csharp_lib(ctx)
    elif lang == 'go':
        _init_go_lib(ctx)
    elif lang == 'java':
        pass
    elif lang == 'nodejs':
        _init_nodejs_lib(ctx)
    elif lang == 'php':
        _init_php_lib(ctx)
    elif lang == 'python':
        pass
    elif lang == 'ruby':
        _init_ruby_lib(ctx)
    else:
        raise Exception('unknown language: {}'.format(lang))

def _init_lang_env(lang, ctx):
    if lang == 'csharp':
        return _init_csharp_env(ctx)
    elif lang == 'go':
        return _init_go_env(ctx)
    elif lang == 'java':
        return _init_java_env(ctx)
    elif lang == 'nodejs':
        return _init_nodejs_env(ctx)
    elif lang == 'php':
        return _init_php_env(ctx)
    elif lang == 'python':
        return _init_python_env(ctx)
    elif lang == 'ruby':
        return _init_ruby_env(ctx)
    else:
        raise Exception('unknown language: {}'.format(lang))

def _write_override_files(ctx):
    filenames = []
    name_dot_version = '{}.{}'.format(ctx.name, ctx.version)
    dv_override_filename = '{}.override1.json'.format(name_dot_version)
    dv_override_filename = os.path.join(ctx.src_dir, dv_override_filename)
    filenames.append(dv_override_filename)
    cmd = 'python generate_default_value_override.py {} --output {}'.format(
            ctx.discovery_doc_filename, dv_override_filename)
    subprocess.check_call(shlex.split(cmd))

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

        mock_discovery_doc_filename = '{}/{}.{}.json'.format(src_dir, name, version)
        cmd = 'python generate_mock_discovery_document.py {} --output {}'.format(
                discovery_doc_filename, mock_discovery_doc_filename)
        subprocess.check_call(shlex.split(cmd))

        cmd = 'python generate_mock_server.py {} --directory {}'.format(mock_discovery_doc_filename, src_dir)
        subprocess.check_call(shlex.split(cmd))

        ctxs.append(Context(name, canonical_name, version, revision,
                mock_discovery_doc_filename, autogen_src_dir, src_dir,
                lib_dir))
    if not ctxs:
        raise Exception('no IDs to test')

    #cmd = './gradlew discoJar'
    #subprocess.check_call(shlex.split(cmd), cwd='toolkit')

    for ctx in ctxs:
        for lang in langs:
            _init_lang_lib(lang, ctx)

    for ctx in ctxs:
        override_filenames = _write_override_files(ctx)
        for lang in langs:
            cmd = ('java -jar discoGen-0.0.5.jar'
                   ' --discovery_doc {}'
                   ' --gapic_yaml {}'
                   ' --overrides {}'
                   ' --output {}').format(ctx.discovery_doc_filename,
                                          _GAPIC_YAML_FILENAMES[lang],
                                          ','.join(override_filenames),
                                          temp_dir)
            subprocess.check_call(shlex.split(cmd))

            cmds = _init_lang_env(lang, ctx)

            cmd = 'python {}.{}.mock.py'.format(ctx.name, ctx.version)
            print(cmd)
            print('Sleeping for 3 seconds, hope the socket is free!')
            time.sleep(3)
            proc = subprocess.Popen(shlex.split(cmd), cwd=ctx.src_dir, stderr=subprocess.PIPE)
            while not proc.stderr.readline():
                pass
            time.sleep(0.1)
            print('Running samples...')
            start = time.time()
            for cmd in cmds:
                print('{:>48} ...'.format(cmd[2]), end='')
                code = subprocess.call(shlex.split(cmd[0]), cwd=cmd[1], stdout=_DEVNULL)
                if code:
                    print(' fail')
                else:
                    print(' ok')
            end = time.time()
            print('Finished in {} seconds'.format(end - start))

            proc.terminate()
            proc.wait()


if __name__ == '__main__':
    main()
