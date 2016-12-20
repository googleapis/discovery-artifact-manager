package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
)

var commentStartRegexp = regexp.MustCompile(`^\s*/\*\*`)
var commentHtmlCode = regexp.MustCompile(`^\s*\*\s*<code>`)
var commentEmptyLine = regexp.MustCompile(`^\s*\*\s*$`)

func checkPanic(e error) {
	if e != nil {
		panic(e)
	}
}

func readFilteredBuffer(path string) (*bytes.Buffer, error) {
	inFile, err := os.Open(path)
	if err != nil {
		return bytes.NewBuffer(nil), err
	}
	defer inFile.Close()

	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	filterOn := false
	var buffer bytes.Buffer

	for scanner.Scan() {
		print := true
		if commentStartRegexp.Match(scanner.Bytes()) {
			filterOn = true
		} else if filterOn {
			if commentHtmlCode.Match(scanner.Bytes()) ||
				commentEmptyLine.Match(scanner.Bytes()) {
				print = false
			} else {
				filterOn = false
			}
		}

		if print {
			buffer.Write(scanner.Bytes())
			buffer.WriteByte('\n')
		}
	}

	return &buffer, scanner.Err()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: cmd file")
		os.Exit(1)
	}
	path := os.Args[1]
	buffer, err := readFilteredBuffer(path)
	checkPanic(err)
	err = ioutil.WriteFile(path, buffer.Bytes(), 0644)
	checkPanic(err)
}
