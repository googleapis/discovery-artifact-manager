// Package csharp implements compile checking for C#.code samples.
package csharp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"discovery-artifact-manager/snippetgen/common/fragment"
)

// Init performs the C#-specific, language-independent
// initialization. This involves setting up a Docker image
// (`languageImageName`) and running as much of its set-up as
// possible. The actual checks will fork off this image to test
// specific APIs. If `force` is set, the image will be created from
// scratch unconditionally; otherwise, `languageImageName` will be
// created only if it doesn't already exist.
func Init(force bool) (string, error) {
	output := &bytes.Buffer{}
	if !force {
		output.Write([]byte("\n# Checking for existing Docker image for csharp\n"))
		cmd := exec.Command("docker", "images")
		out, err := cmd.Output()
		output.Write(out)
		if err != nil {
			return output.String(), err
		}
		if strings.Contains(string(out), languageImageName) {
			return output.String(), nil
		}
	}

	dir, err := ioutil.TempDir("", "compilecheck_setup_csharp")
	if err != nil {
		return output.String(), err
	}

	if err = generateDockerFile(initDockerFile, dir); err != nil {
		return output.String(), err
	}

	if err = generateInitProjectJSON(dir); err != nil {
		return output.String(), err
	}

	if err = generateInitScript(dir); err != nil {
		return output.String(), err
	}

	var runInit string
	output.Write([]byte("\n# Initializing Docker image for csharp\n"))
	if runInit, err = generateRunInit(dir); err != nil {
		return output.String(), fmt.Errorf("generateRunInit: %s", err)
	}

	cmd := exec.Command(runInit)
	cmd.Dir = dir
	out, err := cmd.Output()
	output.Write(out)
	if err != nil {
		return output.String(), fmt.Errorf("running the init command: %s", err)
	}

	return output.String(), nil

}

// Check sets up the C# compile check, satisfying checked.Func.
// It wraps each snippet into its own namespace, allowing them all to compile
// within the sample library.
// A Docker image is created with:
// * mono, which is a .NET runtime and C# compiler.
// * The dotnet CLI, which is a package manager and C# build system.
// * All the namespace-wrapped snippet source files.
// The Docker image is built and executed, which returns the build status as
// a standard exit code from docker.
func Check(files []string, _, tstDir string) (string, error) {
	// Remove old srcs to make space
	if err := os.RemoveAll(tstDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(tstDir, 0750); err != nil {
		return "", err
	}

	if err := copyClasses(files, tstDir); err != nil {
		return "", err
	}

	var libs []string
	var err error
	if libs, err = requiredLibraries(files); err != nil {
		return "", err
	}

	if err := generateCheckProjectJSON(tstDir, libs); err != nil {
		return "", err
	}

	if err := generateBuildScript(tstDir); err != nil {
		return "", err
	}

	var runCheck string
	if runCheck, err = generateRunCheck(tstDir); err != nil {
		return "", err
	}
	if err := generateDockerFile(checkDockerFile, tstDir); err != nil {
		return "", err
	}

	return fmt.Sprintf("# Make sure C# compilation works.\n# This requires docker ('sudo apt-get install docker' on debian/ubuntu)\n%s", runCheck), nil
}

// generateCheckProjectJSON generates the project.json file required by the dotnet cli.
func generateCheckProjectJSON(tstDir string, libs []string) error {
	var libsContent bytes.Buffer
	for _, lib := range libs {
		fmt.Fprintf(&libsContent, "    \"%s\": \"*\",\n", lib)
	}
	content := []byte(
		`{
  "version": "0.1.0-*",

  "dependencies": {
` + libsContent.String() + `
    "Google.Apis.Auth": "*"
  },

  "frameworks": {
    "dnx451": {
      "frameworkAssemblies": {
        "System.Runtime": ""
      }
    }
  }
}
`)
	return ioutil.WriteFile(filepath.Join(tstDir, "project.json"), content, 0755)
}

// generateBuildScript generates a bash script which performs the C# build steps.
// It generates a script that is executed within the docker environment.
func generateBuildScript(tstDir string) error {
	content := []byte(
		`#!/bin/bash
cd /tst
/dotnet restore
/dotnet build
`)
	return ioutil.WriteFile(filepath.Join(tstDir, "build.sh"), content, 0755)
}

// generateRunCheck generates the bash script which builds and runs the Docker image.
// It is named 'RunCheck' as it runs the compile-check.
func generateRunCheck(tstDir string) (string, error) {
	rootName := fmt.Sprintf("%s_%d", imageNameBase, os.Getpid())
	imageName := fmt.Sprintf("%s:latest", rootName)
	containerName := fmt.Sprintf("%s_container", rootName)

	content := []byte(fmt.Sprintf(`#!/bin/bash
cd %s
docker build -t %s .
docker run --name %s -t %s
docker rm -f %s
docker rmi -f %s
`,
		tstDir,
		imageName,
		containerName, imageName,
		containerName,
		imageName))
	path := filepath.Join(tstDir, "RunCheck.sh")
	err := ioutil.WriteFile(path, content, 0755)
	return path, err
}

// generateDockerFile generates the Dockerfile required to perform C# builds.
func generateDockerFile(content string, dir string) error {
	return ioutil.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(content), 0644)
}

// requiredLibs parses file names in `fnames` and returns a list of client libraries needed to
// check those files.
// It returns these library dependencies in the format required by the project.json file.
func requiredLibraries(fnames []string) ([]string, error) {
	type libID struct {
		Name, Version string
	}
	libSet := make(map[libID]bool)
	for _, fname := range fnames {
		p, err := fragment.ParseFileName(fname)
		if err != nil {
			return nil, err
		}
		libSet[libID{
			Name:    p.APIName,
			Version: p.APIVersion,
		}] = true
	}

	libs := make([]string, 0, len(libSet))
	for l := range libSet {
		version := strings.Replace(l.Version, ".", "_", -1)
		libs = append(libs, fmt.Sprintf("Google.Apis.%s.%s", l.Name, version))
	}
	return libs, nil
}

// copyClasses copies C# source code from `files`,
// into cs files, each with a unique package name PKG under dstDir/PKG/ClassName.cs
func copyClasses(files []string, dstDir string) error {
	for i, fname := range files {
		pkgName := fmt.Sprintf("p%d", i)
		if err := copyClass(fname, dstDir, pkgName); err != nil {
			return err
		}
	}
	return nil
}

// copyClass copies a cs source code from file `srcFile`, appending a unique ID to each namespace,
// and writes the content into `dstDir/pkg/ClassName.cs`.
func copyClass(srcFile, dstDir, pkg string) error {
	content, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return err
	}

	pkgDir := filepath.Join(dstDir, pkg)
	dstFile := path.Base(srcFile)

	if err := os.MkdirAll(pkgDir, 0750); err != nil {
		return err
	}

	var dstContent bytes.Buffer
	fmt.Fprintf(&dstContent, "namespace %s {\n", pkg)
	dstContent.Write(content)
	fmt.Fprint(&dstContent, "\n}\n")

	dst := filepath.Join(pkgDir, dstFile)

	return ioutil.WriteFile(dst, dstContent.Bytes(), 0640)
}

// generateRunInit generates the bash script which builds and runs the Docker image.
// It is named 'RunInit' as it runs the initialization code.
func generateRunInit(tstDir string) (string, error) {
	languageEmptyImageName := fmt.Sprintf("%s_empty:latest", languageImageName)
	languageEmptyContainerName := fmt.Sprintf("%s_empty_%d", languageImageName, os.Getpid())

	content := []byte(fmt.Sprintf(`#!/bin/bash
cd %s
docker build -t %s .
docker run --name %s -t %s
docker commit -m "basic dotnet install" %s %s:latest
docker rm -f %s
docker rmi -f %s
`,
		tstDir,
		languageEmptyImageName,
		languageEmptyContainerName, languageEmptyImageName,
		languageEmptyContainerName, languageImageName,
		languageEmptyContainerName,
		languageEmptyImageName))
	path := filepath.Join(tstDir, "RunInit.sh")
	err := ioutil.WriteFile(path, content, 0755)
	return path, err
}

// generateInitProjectJSON generates the project.json file required by
// the dotnet cli. It is used in Initialization to prepare the parts
// of the Docker container that are independent of the APIs being
// checked themselves.
func generateInitProjectJSON(dir string) error {
	content := []byte(
		`{
  "version": "0.1.0-*",

  "dependencies": {
    "Google.Apis.Auth": "*"
   },

  "frameworks": {
    "dnx451": {
      "frameworkAssemblies": {
        "System.Runtime": ""
      }
    }
  }
}
`)
	return ioutil.WriteFile(filepath.Join(dir, "project.json"), content, 0755)
}

// generateInitScript generates a bash script which performs as much
// preparation of the C# environment as possible.  It generates a
// script that is executed within the Docker environment.
func generateInitScript(tstDir string) error {
	content := []byte(
		`#!/bin/bash
cd /tst
/dotnet restore
/dotnet build
`)
	return ioutil.WriteFile(filepath.Join(tstDir, "init.sh"), content, 0755)
}

var (
	// initDockerFile is contents of the initial DockerFile used
	// to set up the `languageImageName` Docker image to be used
	// for all C# samples.
	initDockerFile = `FROM ubuntu:xenial
RUN apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 3FA7E0328081BFF6A14DA29AA6A19B38D3D831EF
RUN echo "deb http://download.mono-project.com/repo/debian wheezy/snapshots/4.2.3.4 main" > /etc/apt/sources.list.d/mono-xamarin.list \
    && echo "deb http://download.mono-project.com/repo/debian wheezy-apache24-compat main" >> /etc/apt/sources.list.d/mono-xamarin.list \
    && echo "deb http://download.mono-project.com/repo/debian wheezy-libjpeg62-compat main" >> /etc/apt/sources.list.d/mono-xamarin.list \
    && apt-get update \
    && apt-get install -y mono-devel libunwind-dev curl libcurl3-dev
RUN curl -O https://dotnetcli.blob.core.windows.net/dotnet/preview/Binaries/Latest/dotnet-dev-ubuntu.16.04-x64.latest.tar.gz
RUN tar -xf dotnet-dev-ubuntu.16.04-x64.latest.tar.gz
ADD . /tst
ENTRYPOINT /tst/init.sh

`

	// checkDockerFile is contents of the DockerFile used to
	// modify the `languageImageName` Docker image for use when
	// checking a specific API.
	checkDockerFile string

	// imageNameBase is the root name of the Docker imagees we create for C#.
	imageNameBase = "compilecheck_csharp"

	// languageImageName is the name of the language-specific,
	// API-independent Docker image that is the basis for all the
	// Docker images that perform the actuall compile checks.
	languageImageName string
)

// init constructs the derived image names and DockerFile contents.
func init() {
	languageImageName = fmt.Sprintf("%s_master", imageNameBase)

	checkDockerFile = fmt.Sprintf(`FROM %s
ADD . /tst
ENTRYPOINT /tst/build.sh
`, languageImageName)
}
