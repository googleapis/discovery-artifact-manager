package clientlib

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadURL(t *testing.T) {
	name := "service"
	version := "v1"
	expect := map[string]string{
		"Java":    "https://developers.google.com/resources/api-libraries/download/service/v1/java",
		"Node.js": "https://github.com/google/google-api-nodejs-client/archive/master.zip",
		"PHP":     "https://github.com/google/google-api-php-client-services/archive/master.zip",
		"Python":  "https://github.com/google/google-api-python-client/archive/master.zip",
		"Ruby":    "https://github.com/google/google-api-ruby-client/archive/master.zip",
	}
	for key, val := range expect {
		got, err := DownloadURL(key, name, version)
		if err != nil {
			t.Errorf("unexpected error")
		}
		if val != got {
			t.Errorf("for %q expected %q, but got %q", key, expect, got)
		}
	}

	_, err := DownloadURL("foolang", name, version)
	if err == nil {
		t.Errorf("expected error for invalid language, got nil")
	}
}

func TestLandingPage(t *testing.T) {
	name := "service"
	version := "v1"
	expect := map[string]string{
		"Java":    "https://developers.google.com/api-client-library/java/apis/service/v1",
		".NET":    "https://developers.google.com/api-client-library/dotnet/apis/service/v1",
		"PHP":     "https://github.com/google/google-api-php-client-services",
		"Python":  "https://github.com/google/google-api-python-client",
		"Ruby":    "https://github.com/google/google-api-ruby-client",
		"Go":      "https://github.com/google/google-api-go-client",
		"Node.js": "https://github.com/google/google-api-nodejs-client",
	}
	for key, val := range expect {
		got, err := LandingPage(key, name, version)
		if err != nil {
			t.Errorf("unexpected error")
		}
		if val != got {
			t.Errorf("for %q expected %q, but got %q", key, expect, got)
		}
	}

	_, err := LandingPage("foolang", name, version)
	if err == nil {
		t.Errorf("expected error for invalid language, got nil")
	}
}

func TestUnzip(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"abc":     "abc content",
		"def/xyz": "other content",
	}

	var buf MemBuffer
	zwr := zip.NewWriter(&buf)
	for name, content := range files {
		wr, err := zwr.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = wr.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zwr.Close(); err != nil {
		t.Fatal(err)
	}

	sz, err := buf.FinishWrite()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err = unzip(&buf, sz, tmpDir); err != nil {
		t.Error(err)
	}
	for name, content := range files {
		gotBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, name))
		if err != nil {
			t.Error(err)
		}
		if got := string(gotBytes); got != content {
			t.Errorf("file %q, got %q, want %q", name, got, content)
		}
	}
}

func TestCopyToFile(t *testing.T) {
	t.Parallel()

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	fname := filepath.Join(tmpDir, "myfile")

	const content = "foobarzipzap"
	rd := ioutil.NopCloser(bytes.NewBufferString(content))
	if err = copyToFile(rd, fname); err != nil {
		t.Error(err)
	}

	gotBytes, err := ioutil.ReadFile(fname)
	if err != nil {
		t.Error(err)
	}
	if got := string(gotBytes); got != content {
		t.Errorf("expected content %q, want %q", got, content)
	}
}
