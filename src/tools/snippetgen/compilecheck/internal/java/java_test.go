package java

import (
	"fmt"
	"discovery-artifact-manager/tools/snippetgen/common/clientlib"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyClass(t *testing.T) {
	t.Parallel()
	const pkgName = "mypackage"
	const className = "FooBar"

	var javaFile = fmt.Sprintf(`
public class %s {
  private int x;
}
`, className)

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcFile := filepath.Join(tempDir, "src")
	if err = ioutil.WriteFile(srcFile, []byte(javaFile), 0644); err != nil {
		t.Fatal(err)
	}
	dstFile, err := copyClass(srcFile, tempDir, pkgName)
	if err != nil {
		t.Error(err)
	}

	if exp := filepath.Join(tempDir, pkgName, className+".java"); exp != dstFile {
		t.Errorf("wrong destination file %q, want %q", dstFile, exp)
	}

	contentBytes, err := ioutil.ReadFile(dstFile)
	if err != nil {
		t.Error(err)
	}
	content := string(contentBytes)
	if pkgdecl := fmt.Sprintf("package %s;", pkgName); !strings.Contains(content, pkgdecl) {
		t.Errorf("cannot find proper package declaration %q in content:\n%s", pkgdecl, content)
	}
	if !strings.Contains(content, javaFile) {
		t.Errorf("cannot find copied content\n%s\nin file: %s", javaFile, content)
	}
}

func TestJavacOpt(t *testing.T) {
	tsts := []struct {
		jarFiles, testFiles []string
		want                string
	}{
		{
			jarFiles:  []string{"/path/to/jar.jar", "another/jarrr.jar"},
			testFiles: []string{"/path/to/Class.java", "another/Clazz.java"},
			want:      "-cp '/path/to/jar.jar:another/jarrr.jar' /path/to/Class.java another/Clazz.java",
		},
	}

	for _, tst := range tsts {
		got := string(generateJavacOpt(tst.jarFiles, tst.testFiles))
		if got != tst.want {
			t.Errorf("javacOpt(%q, %q) = %q, want %q", tst.jarFiles, tst.testFiles, got, tst.want)
		}
	}
}

func TestRequiredLibs(t *testing.T) {
	fnames := []string{
		"path/to/pubsub/v1/1234/pubsub.projects.subscriptions.acknowledge.frag.java",
		"container/v1/20150603/container.projects.zones.clusters.create.frag.java",
		"container/v1/20150603/container.projects.zones.clusters.delete.frag.java",
		"translate/v2/20160217/language.detections.list.frag.java",
	}
	expect := map[clientlib.Lib]bool{
		{"pubsub", "https://developers.google.com/resources/api-libraries/download/pubsub/v1/java"}:       true,
		{"container", "https://developers.google.com/resources/api-libraries/download/container/v1/java"}: true,
		{"translate", "https://developers.google.com/resources/api-libraries/download/translate/v2/java"}: true,
	}

	libs, err := requiredLibraries(fnames)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	for _, l := range libs {
		if _, ok := expect[l]; !ok {
			t.Errorf("unexpected client lib: %v", l)
		}
		delete(expect, l)
	}

	for l := range expect {
		t.Errorf("expected client lib but not found: %v", l)
	}
}
