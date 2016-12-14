package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"

	"gapi-cmds/src/common/errorlist"
	"gapi-cmds/src/snippetgen/compilecheck/internal/checker"
)

func main() {
	libDir := flag.String("lib",
		filepath.Join(os.TempDir(), "discovery", "lib"),
		"directory in which to put client libraries that we are testing against")
	tstDir := flag.String("tst",
		filepath.Join(os.TempDir(), "discovery", "tst"),
		"directory in which to put the test program")

	init := flag.Bool("init", true, "Whether to perform language-specific, API-independent initializations")
	cleanInit := flag.Bool("clean", false, "Whether to perform a clean init even if it doesn't appear necessary. This flag has no effect if --init=false.")
	runCheck := flag.Bool("check", true, "Whether to perform the compile check itself (assumes language initialization has been done via --init")

	profile := flag.String("pprof", "", "file to place CPU profiling")
	flag.Parse()

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *init {
		if out, err := checker.Init(*cleanInit); err != nil {
			log.Fatalf("%s\n\nOutput:\n\n%s", err, out)
		}
	}

	if !(*runCheck) {
		os.Exit(0)
	}

	files, err := getAllFiles(flag.Args())
	if err != nil {
		log.Fatal(err)
	}

	langs, err := splitFilenamesByLanguage(files)
	if err != nil {
		log.Fatal(err)
	}

	var checkCmds []string
	var errlist errorlist.Errors
	for _, ext := range sortedKeys(langs) {
		chk, ok := checker.Checkers[ext]
		if !ok {
			errlist.Add(fmt.Errorf("unknown file extension: %s", ext))
			continue
		}

		var lib, tst string
		lib, err = filepath.Abs(filepath.Join(*libDir, ext))
		if err != nil {
			log.Fatal(err)
		}
		tst, err = filepath.Abs(filepath.Join(*tstDir, ext))
		if err != nil {
			log.Fatal(err)
		}

		var checkCmd string
		if checkCmd, err = chk.Fn(langs[ext], lib, tst); err != nil {
			log.Fatal(err)
		}
		checkCmds = append(checkCmds,
			fmt.Sprintf("echo;echo === Checking %q\n%s", ext, checkCmd))
	}

	fmt.Print("do_compile_check() {\nlocal status=0\n")
	for _, instr := range checkCmds {
		fmt.Printf("%s || status=$?\n", strings.TrimSpace(instr))
	}
	fmt.Print("return $status\n}\ndo_compile_check\n")

	if err = errlist.Error(); err != nil {
		log.Fatal(err)
	}
}

// getAllFiles recursively list all files under `args`. For each element, if the element is a
// normal file, the file is simply included.
// If the element is a directory, files under the directory are included recursively.
// getAllFiles does not follow symlinks.
func getAllFiles(args []string) ([]string, error) {
	var files []string
	for _, arg := range args {
		err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// splitFilenamesByLanguage groups different files in `files` by file extensions.
func splitFilenamesByLanguage(files []string) (map[string][]string, error) {
	langs := map[string][]string{}
	for _, fname := range files {
		p := strings.LastIndexByte(fname, '.')
		if p < 0 {
			return nil, fmt.Errorf("cannot get file extension: %s", fname)
		}
		ext := fname[p+1:]
		langs[ext] = append(langs[ext], fname)
	}
	return langs, nil
}

// sortedKeys returns a slice containing sorted keys of `m`.
func sortedKeys(m map[string][]string) []string {
	ar := make([]string, 0, len(m))
	for k := range m {
		ar = append(ar, k)
	}
	sort.Strings(ar)
	return ar
}
