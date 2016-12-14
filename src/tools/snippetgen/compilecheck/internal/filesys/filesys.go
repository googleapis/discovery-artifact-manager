// Package filesys contains abstractions and mocks of the file system, useful for testing.
package filesys

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// FS is the union of Opener and Creator
type FS interface {
	Opener
	Creator
}

// Opener abstracts reading from the filesystem. Used for testing.
type Opener interface {
	// Open opens file `s` for reading.
	Open(string) (io.ReadCloser, error)

	// ReadFile reads the named file and returns the contents and any error encountered.
	ReadFile(string) ([]byte, error)
}

// Creator abstracts file creation from the filesystem. Used for testing.
type Creator interface {
	// Create creates file `s` for writing. If the file already exists, it is truncated.
	Create(string) (io.WriteCloser, error)

	// WriteFile writes `data` to `file`. If the file doesn't exist, WriteFile creates it with `perm`.
	WriteFile(file string, data []byte, perm os.FileMode) error
}

// OS implements Opener and Creator by delegating to os.
type OS struct{}

// Open implements Opener.Open
func (OS) Open(s string) (io.ReadCloser, error) {
	return os.Open(s)
}

// ReadFile implements Opener.ReadFile
func (OS) ReadFile(s string) ([]byte, error) {
	return ioutil.ReadFile(s)
}

// Create implements Creator.Create.
func (OS) Create(s string) (io.WriteCloser, error) {
	return os.Create(s)
}

// WriteFile implements Creator.WriteFile.
func (OS) WriteFile(file string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(file, data, perm)
}

// MapFS is an in-memory implementation of FS, where all "files" are map entries,
// useful for testing.
type MapFS map[string]string

// Open returns a reader to a map entry. If the key `s` is not found, it returns an *os.PathError.
func (fs MapFS) Open(s string) (io.ReadCloser, error) {
	content, ok := fs[s]
	if !ok {
		return nil, &os.PathError{
			Op:   "open",
			Path: s,
			Err:  os.ErrNotExist,
		}
	}
	return ioutil.NopCloser(strings.NewReader(content)), nil
}

// ReadFile returns the content of a map entry. If the key is not found,
// it returns an *os.PathError.
func (fs MapFS) ReadFile(s string) ([]byte, error) {
	content, ok := fs[s]
	if !ok {
		return nil, &os.PathError{
			Op:   "open",
			Path: s,
			Err:  os.ErrNotExist,
		}
	}
	return []byte(content), nil
}

// Create implements Creator.Create. To emulate the actual filesystem, writes to the returned
// WriteCloser are not visible until the WriteCloser is Close()d.
func (fs MapFS) Create(s string) (io.WriteCloser, error) {
	fs[s] = ""
	return &creatorWriter{
		fs:  fs,
		key: s,
	}, nil
}

// WriteFile implements Creator.WriteFile.
func (fs MapFS) WriteFile(file string, data []byte, _ os.FileMode) error {
	d := string(data)
	fs[file] = d
	return nil
}

// creatorWriter is the io.WriteCloser implementation for MapCreator. It attempts to catch improper
// uses of file descriptors; for more details, see docs for individual methods.
type creatorWriter struct {
	bytes.Buffer
	fs     MapFS
	key    string
	closed bool
}

// Write implements io.Writer. As would be the case in normal files, writing after a close is an
// error. Writes are not visible to the MapCreator until the writer has been closed.
func (w *creatorWriter) Write(b []byte) (int, error) {
	if w.closed {
		return 0, errors.New("file is closed")
	}
	return w.Buffer.Write(b)
}

// Close implements io.Closer, allowing written data to be retrieved from MapCreator.
// Double closing is an error.
func (w *creatorWriter) Close() error {
	if w.closed {
		return errors.New("already closed")
	}
	w.closed = true
	w.fs[w.key] = w.Buffer.String()
	return nil
}
