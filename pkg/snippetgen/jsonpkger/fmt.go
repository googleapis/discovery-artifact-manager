package main

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/googleapis/discovery-artifact-manager/pkg/snippetgen/common/fragment"
)

var formatters = []struct {
	langName string
	f        func([]*fragment.CodeFragment) error
}{
	{"Go", func(frags []*fragment.CodeFragment) error {
		for _, frag := range frags {
			fmt, err := format.Source([]byte(frag.Fragment))
			if err != nil {
				return err
			}
			frag.Fragment = string(fmt)
		}
		return nil
	}},
	{"Java", func(frags []*fragment.CodeFragment) error {
		// foo
		tmpDir, err := ioutil.TempDir("", "jsonpkger-java-format")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		fnames := make([]string, 0, len(frags))
		for i, frag := range frags {
			fname := filepath.Join(tmpDir, fmt.Sprintf("p%d.java", i))
			fnames = append(fnames, fname)
			if err := ioutil.WriteFile(fname, []byte(frag.Fragment), 0650); err != nil {
				return err
			}
		}

		// google-java-format comes standard in workstations. No need to check/install.
		cmdArg := append([]string{"-i"}, fnames...)
		if err := exec.Command("google-java-format", cmdArg...).Run(); err != nil {
			return err
		}

		for i, fname := range fnames {
			src, err := ioutil.ReadFile(fname)
			if err != nil {
				return err
			}
			frags[i].Fragment = string(src)
		}
		return nil
	}},
}

func formatFragments(fragments fragmentLanguageMap) error {
	fragsByLang := make(map[string][]*fragment.CodeFragment)
	for _, langFrags := range fragments {
		for lang, frag := range langFrags {
			fragsByLang[lang] = append(fragsByLang[lang], frag)
		}
	}
	for _, fmter := range formatters {
		if err := fmter.f(fragsByLang[fmter.langName]); err != nil {
			return err
		}
	}
	return nil
}
