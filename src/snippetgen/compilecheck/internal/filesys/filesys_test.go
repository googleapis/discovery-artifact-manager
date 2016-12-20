package filesys

import (
	"io/ioutil"
	"os"
	"testing"
)

// Compile-time check that OS and MapFS implement the FS interface.
var _ FS = OS{}
var _ FS = MapFS{}

func TestMapOpener(t *testing.T) {
	opener := MapFS{
		"abc": "def",
	}

	tests := []struct {
		name, content string
		err           bool
	}{
		{"abc", "def", false},
		{"xyz", "", true},
	}

	for _, test := range tests {
		rd, err := opener.Open(test.name)
		var content string
		if rd != nil {
			c, err2 := ioutil.ReadAll(rd)
			if err2 != nil {
				t.Fatalf("unexpected error: %s", err2)
			}
			content = string(c)
		}
		switch {
		case test.err && !os.IsNotExist(err):
			t.Errorf("%s: got error %q, want not found", test.name, err)
		case err != nil && !test.err:
			t.Errorf("%s: unexpected error while opening: %s", test.name, err)
		case content != test.content:
			t.Errorf("%s: got %q, want %q", test.name, content, test.content)
		}
	}
}

func TestMapCreator(t *testing.T) {
	const fname = "abc"
	const content = "stuff"

	creator := MapFS{}
	wr, err := creator.Create(fname)
	if err != nil {
		t.Fatal(err)
	}
	wr.Write([]byte(content))

	switch got, ok := creator[fname]; {
	case !ok:
		t.Errorf("%s: doesn't exist, expected to be empty string", fname)
	case got != "":
		t.Errorf("%s: got %q, expected to be empty string since writer was not closed", fname, got)
	}

	if err = wr.Close(); err != nil {
		t.Fatal(err)
	}
	if got := creator[fname]; got != content {
		t.Errorf("%s: got %q, expected %q", fname, got, content)
	}
}

func TestMapCreatorErrors(t *testing.T) {
	creator := MapFS{}
	wr, err := creator.Create("abc")
	if err != nil {
		t.Fatal(err)
	}
	if err = wr.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err = wr.Write([]byte("foo")); err == nil {
		t.Errorf("writer already closed, writes should error")
	}
	if err = wr.Close(); err == nil {
		t.Errorf("writer already closed, further closes should error")
	}
}

func TestWriteFile(t *testing.T) {
	const fname, content = "abc", "def"

	creator := MapFS{}
	if err := creator.WriteFile(fname, []byte(content), 0750); err != nil {
		t.Fatal(err)
	}
	if data := creator[fname]; data != content {
		t.Errorf("%s: wrong content, got %q, want %q", fname, data, content)
	}
}
