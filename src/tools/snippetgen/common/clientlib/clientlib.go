// Package clientlib provides utility functions to help with downloading client libraries.
package clientlib

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"gapi-cmds/src/common/errorlist"
)

// Lib represents a downloadable client library.
type Lib struct {
	Name, URL string
}

// DownloadURL returns the URL of the client library archive for the given
// language and service name/version. If an unknown language is provided, an
// error is returned.
func DownloadURL(language, name, version string) (string, error) {
	switch language {
	case "Java":
		return fmt.Sprintf("https://developers.google.com/resources/api-libraries/download/%s/%s/java", name, version), nil
	case "Node.js":
		return "https://github.com/google/google-api-nodejs-client/archive/master.zip", nil
	case "PHP":
		return "https://github.com/google/google-api-php-client-services/archive/master.zip", nil
	case "Python":
		return "https://github.com/google/google-api-python-client/archive/master.zip", nil
	case "Ruby":
		return "https://github.com/google/google-api-ruby-client/archive/master.zip", nil
	default: // The other languages don't require or use an archive URL.
		return "", fmt.Errorf("unexpected language: %q", language)
	}
}

// LandingPage returns the client library landing page for the given language
// and service name/version. If an unknown language is provided, an error is
// returned.
func LandingPage(language, name, version string) (string, error) {
	switch language {
	case "Java":
		return fmt.Sprintf("https://developers.google.com/api-client-library/java/apis/%s/%s", name, version), nil
	case ".NET":
		return fmt.Sprintf("https://developers.google.com/api-client-library/dotnet/apis/%s/%s", name, version), nil
	case "PHP":
		return "https://github.com/google/google-api-php-client-services", nil
	case "Python":
		return "https://github.com/google/google-api-python-client", nil
	case "Ruby":
		return "https://github.com/google/google-api-ruby-client", nil
	case "Go":
		return "https://github.com/google/google-api-go-client", nil
	case "Node.js":
		return "https://github.com/google/google-api-nodejs-client", nil
	default:
		return "", fmt.Errorf("unexpected language: %q", language)
	}
}

// DownloadUnzipIfMissing downloads libraries `libs` into directory `dst` if the libraries don't
// already exist.
//
// Library `l` is considered to "exist" if directory "dstDir/{l.Name}" exists.
// If the library doesn't yet exist, the content of l.URL is unzipped into "dstDir/{l.Name}".
func DownloadUnzipIfMissing(libs []Lib, dstDir string) error {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var errlist errorlist.Errors

	wg.Add(len(libs))
	for _, lib := range libs {
		go func(l Lib) {
			defer wg.Done()
			dst := filepath.Join(dstDir, l.Name)
			if _, err := os.Stat(dst); err == nil {
				return
			} else if !os.IsNotExist(err) {
				mu.Lock()
				defer mu.Unlock()
				errlist.Add(err)
				return
			}
			if err := DownloadUnzip(l.URL, dst, new(MemBuffer)); err != nil {
				mu.Lock()
				defer mu.Unlock()
				errlist.Add(err)
			}
		}(lib)
	}

	wg.Wait()
	return errlist.Error()
}

// Buffer is the interface to provide DownloadUnzip with buffer space.
type Buffer interface {
	io.Writer

	// FinishWrite is called after all writes and before any read, giving Buffer a chance to prepare
	// for reading. It should return the amount of data it has read, in bytes,
	// and any errors encountered.
	FinishWrite() (int64, error)

	io.ReaderAt
}

// DownloadUnzip downloads and unzips client libraries.
//
// `url` is assumed to be a zip archive containing things we want.
// `dst` is the directory that the content should be unzipped to.
// `buf` is used to as buffer space for unzipping.
//
// If non-nil error is returned, there is no guarantee as to which files, if any, were written
// to `dst`.
//
// DownloadUnzip is safe to be called in parallel. However, races can happen if two invocations
// want to write to the same files.
func DownloadUnzip(url, dst string, buf Buffer) error {
	if err := download(url, buf); err != nil {
		return err
	}
	sz, err := buf.FinishWrite()
	if err != nil {
		return err
	}
	return unzip(buf, sz, dst)
}

// download downloads from `url`, placing content into `wr`.
func download(url string, wr io.Writer) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(wr, resp.Body)
	return err
}

// unzip unzips zip archive from `reader`, which must have size `sz`, placing contents into
// directory `dst`.
func unzip(reader io.ReaderAt, sz int64, dst string) error {
	zrd, err := zip.NewReader(reader, sz)
	if err != nil {
		return err
	}
	for _, file := range zrd.File {
		if file.Mode().IsDir() {
			continue
		}
		dstFile := filepath.Join(dst, filepath.FromSlash(file.Name))
		rd, err := file.Open()
		if err != nil {
			return err
		}
		if err = copyToFile(rd, dstFile); err != nil {
			return err
		}
	}
	return nil
}

// copyToFile copies the `src` into file `dstFile`.
//
// `src` must already be opened. The file named `dstFile` is created, if it doesn't exist.
// Both are closed by this function.
func copyToFile(src io.ReadCloser, dstFile string) error {
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(dstFile), 0755); err != nil {
		return err
	}
	f, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, src)
	return err
}

// MemBuffer implements Buffer in-memory.
type MemBuffer struct {
	bytes.Buffer
	*bytes.Reader
}

// FinishWrite prepares b for reading. Always return nil error.
func (b *MemBuffer) FinishWrite() (int64, error) {
	b.Reader = bytes.NewReader(b.Buffer.Bytes())
	return int64(b.Reader.Len()), nil
}

// Reset resets MemBuffer, allowing the allocation to be reused.
func (b *MemBuffer) Reset() {
	b.Reader = nil
	b.Buffer.Reset()
}
