// Package gcs allows users to run the gsutil command to upload data
// to Google Cloud storage.
package gcs

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

// GCS handles uploading fragments to GCS.
type GCS struct {
	gsutil string
}

// bucketPrefix is the path prefix that indicates a location in GCS.
const bucketPrefix = "gs://"

// Init initializes 'gcs' with the given path 'gsutilPath' to the
// "gsutil" program on the client's machine. It
// returns an error if it cannot find the gsutil command.
func New(gsutilPath string) (*GCS, error) {
	gsutil, err := exec.LookPath(gsutilPath)
	if err != nil {
		return nil, err
	}
	return &GCS{gsutil: gsutil}, nil
}

// TransferTree initiates a gsutil recursive upload of the contents of
// the directory tree rooted at the 'src' path to the tree rooted at
// tge 'dst' path, and returns the output of the command. One or both
// of these paths can begin with the "gs://" prefix to specify a GCS
// location. Transfer to GCS are marked with content-type
// "application/json", and are made world-readable iff
// 'publiclyReadable' is set. Failures cause an error (with the
// command output embedded) to be returned.
func (gcs *GCS) TransferTree(src, dst string, publiclyReadable bool) (string, error) {
	acl := ""
	if publiclyReadable {
		acl = "-a public-read"
	}
	args := fmt.Sprintf("-h Content-Type:application/json -m cp %s -r %s/* %s/", acl, src, dst)
	cmd := exec.Command(gcs.gsutil, strings.Fields(args)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("gsutil error: %s\nOutput: %s\n", err, output)
	}

	return string(output), nil
}

// IsGCS returns true iff 'filePath' indicates a path on GCS.
func IsGCS(filePath string) bool {
	return strings.HasPrefix(filePath, bucketPrefix)
}

// ListTree returns all the subdirectories specified in `dirList` that
// exist as top-level subdirectories under `root`. If none of these
// exist but `root` exists, it returns an empty list. If `root` also
// does not exist, it returns an error.
func (gcs *GCS) ListTree(root string, dirList []string) ([]string, error) {
	dirs := []string{}

	if IsGCS(root) {
		var contents bytes.Buffer
		for _, dir := range dirList {
			d := strings.Join([]string{root, dir}, "/")
			cmd := exec.Command(gcs.gsutil, strings.Fields(fmt.Sprintf("-q ls %s", d))...)

			cmd.Stdout = &contents
			cmd.Run()

			if contents.Len() != 0 {
				newDirs, err := scanGCSTree(&contents)
				dirs = append(dirs, newDirs...)
				if err != nil {
					return dirs, err
				}
			}
		}
		if len(dirs) > 0 {
			return dirs, nil
		}

		cmd := exec.Command(gcs.gsutil, strings.Fields(fmt.Sprintf("-q ls %s", root))...)
		cmd.Stdout = &contents
		cmd.Run()
		if contents.Len() == 0 {
			return nil, fmt.Errorf("location does not exist: %q", root)
		}
		return nil, nil
	}

	for _, dir := range dirList {
		fullDir := path.Join(root, dir)
		entries, err := ioutil.ReadDir(fullDir)
		if err != nil {
			return nil, fmt.Errorf("could not read local directory %q", fullDir)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, path.Join(fullDir, entry.Name()))
			}
		}
	}

	return dirs, nil
}

// scanGCSTree parses a file listing in 'contents' as returned by a GCS
// "ls" command and returns a list of all the files.
func scanGCSTree(contents *bytes.Buffer) ([]string, error) {
	scanner := bufio.NewScanner(contents)
	var tree []string
	for scanner.Scan() {
		trimmed := bytes.TrimSpace(scanner.Bytes())
		if len(trimmed) > 0 && !bytes.HasSuffix(trimmed, []byte(":")) {
			tree = append(tree, string(bytes.TrimSuffix(trimmed, []byte("/"))))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return tree, nil
}

// TransferTreePaths transfers all the paths in `trees` rooted at
// `src` to the same paths rooted at `dst`.
func (gcs *GCS) TransferTreePaths(src, dst string, publiclyReadable bool, trees []string) (string, error) {
	if len(trees) == 0 {
		return gcs.TransferTree(src, dst, publiclyReadable)
	}
	var output bytes.Buffer
	for _, srcPath := range trees {
		var dstPath string
		if IsGCS(dst) {
			dstPath = fmt.Sprintf("%s/%s", dst, strings.TrimPrefix(strings.TrimPrefix(srcPath, src), "/"))
		} else {
			dstPath = path.Join(dst, strings.TrimPrefix(srcPath, src))
		}
		if err := os.MkdirAll(dstPath, 0777); err != nil {
			return "", err
		}
		cmdOut, err := gcs.TransferTree(srcPath, dstPath, publiclyReadable)
		output.WriteString(cmdOut)
		if err != nil {
			return output.String(), fmt.Errorf("could not transfer tree: %q -> %q: %s", srcPath, dstPath, err)
		}
	}
	return output.String(), nil
}
