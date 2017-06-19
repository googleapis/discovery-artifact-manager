package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"

	"discovery-artifact-manager/common/environment"
	"discovery-artifact-manager/common/errorlist"
)

// discoURL specifies a URL for the live Discovery service index.
const discoURL = "https://www.googleapis.com/discovery/v1/apis"

type apiInfo struct {
	Name, Version, DiscoveryRestURL string
}

type apiIndex struct {
	Items []apiInfo
}

// UpdateDiscos updates local Discovery doc files for all APIs indexed by the live Discovery
// service, in a top-level directory 'discoveries', which must exist; and returns the absolute
// `names` of updated files.
func UpdateDiscos() (names []string, err error) {
	absolutePath, dir, filepaths, err := readDiscoCache()
	if err != nil {
		return
	}

	indexData, err := readDiscoIndex()
	if err != nil {
		return
	}

	updated, errs := writeDiscoCache(indexData, absolutePath, dir)

	cleanDiscoCache(absolutePath, filepaths, updated, &errs)
	err = errs.Error()
	if err != nil {
		return
	}

	names = make([]string, 0, len(updated))
	for filename, _ := range updated {
		path := path.Join(absolutePath, filename)
		names = append(names, path)
	}
	return
}

// readDiscoCache returns the `absolutePath` to the top-level 'discoveries' directory along with its
// `directory` attributes and those of all discovery `filepaths` therein. Note that the discovery
// index file, index.json, is excluded from `files`.
func readDiscoCache() (absolutePath string, directory os.FileInfo, filepaths []string, err error) {
	root, err := environment.RepoRoot()
	if err != nil {
		err = fmt.Errorf("Error finding repository root directory: %v", err)
		return
	}
	absolutePath = path.Join(root, "discoveries")
	directory, err = os.Stat(absolutePath)
	if os.IsNotExist(err) {
		err = fmt.Errorf("Error finding path for Discovery docs: %v", absolutePath)
		return
	}
	globPath := path.Join(absolutePath, "*.json")
	filepaths, err = filepath.Glob(path.Join(absolutePath, "*.json"))
	if err != nil {
		err = fmt.Errorf("Error globbing path for Discovery docs: %v", globPath)
	}
	// Remove "index.json" from files, as it's not a Discovery document.
	for i := 0; i < len(filepaths); i += 1 {
		_, filename := filepath.Split(filepaths[i])
		fmt.Println(filename, filename == "index.json")
		if filename == "index.json" {
			filepaths = append(filepaths[:i], filepaths[i+1:]...)
			break
		}
	}
	return
}

// readDiscoIndex returns the index returned by the live Discovery service as a JSON byte array.
func readDiscoIndex() (indexData []byte, err error) {
	fmt.Printf("Fetching Discovery doc index from %v ...\n", discoURL)
	client := &http.Client{}
	request, err := http.NewRequest("GET", discoURL, nil)
	if err != nil {
		err = fmt.Errorf("Error forming request for Discovery doc index: %v", err)
		return
	}
	// Use extra-Google IP header (RFC 5737 TEST-NET) to limit index results to public APIs
	request.Header.Add("X-User-IP", "192.0.2.0")
	response, err := client.Do(request)
	if err != nil {
		err = fmt.Errorf("Error fetching Discovery doc index: %v", err)
		return
	}
	defer response.Body.Close()
	indexData, err = ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("Error reading Discovery doc index: %v", err)
		return
	}
	return
}

// writeDiscoCache updates (creates or replaces) Discovery doc files in the top-level 'discoveries'
// directory (given its `absolutePath` and `directory` attributes) as needed to update descriptions
// of APIs in `indexData` (assumed not to contain duplicates). It returns a map containing `updated`
// file basenames corresponding to live APIs, and accumulates `errors` from all updates.
func writeDiscoCache(indexData []byte, absolutePath string, directory os.FileInfo) (updated map[string]bool, errors errorlist.Errors) {
	fmt.Printf("Updating local Discovery docs in %v ...\n", absolutePath)
	// Make Discovery doc file permissions like parent directory (no execute)
	perm := directory.Mode() & 0666

	fmt.Println("Parsing and writing Discovery doc index ...")
	index := &apiIndex{}
	err := json.Unmarshal(indexData, index)
	if err != nil {
		err = fmt.Errorf("Error parsing Discovery doc index: %v", err)
		errors.Add(err)
		return
	}
	size := len(index.Items)

	path := path.Join(absolutePath, "index.json")
	if err := ioutil.WriteFile(path, indexData, perm); err != nil {
		err = fmt.Errorf("Error writing Discovery index to %v: %v", path, err)
		errors.Add(err)
	}

	var collect sync.WaitGroup
	errChan := make(chan error, size)
	collect.Add(1)
	go func() {
		defer collect.Done()
		for err := range errChan {
			fmt.Println(err)
			errors.Add(err)
		}
	}()

	updated = make(map[string]bool, size)
	updateChan := make(chan string, size)
	collect.Add(1)
	go func() {
		defer collect.Done()
		for file := range updateChan {
			updated[file] = true
		}
	}()

	var update sync.WaitGroup
	for _, api := range index.Items {
		update.Add(1)
		go func(api apiInfo) {
			defer update.Done()
			if err := UpdateAPI(api, absolutePath, perm, updateChan); err != nil {
				errChan <- fmt.Errorf("Error updating %v %v: %v", api.Name, api.Version, err)
			}
		}(api)
	}
	update.Wait()
	close(errChan)
	close(updateChan)
	collect.Wait()
	return
}

// cleanDiscoCache deletes those `filepaths` in the top-level 'discoveries' directory at
// `absolutePath` whose names do not appear in the map of `updated` files, and accumulates any
// further `errors`.
func cleanDiscoCache(absolutePath string, filepaths []string, updated map[string]bool, errors *errorlist.Errors) {
	for _, path := range filepaths {
		_, filename := filepath.Split(path)
		if !updated[filename] {
			path = filepath.Join(absolutePath, filename)
			if err := os.Remove(path); err != nil {
				errors.Add(fmt.Errorf("Error deleting expired Discovery doc %v: %v", path, err))
			}
		}
	}
}

// UpdateAPI reads the Discovery doc for an `API` indexed by the live Discovery service and updates
// the corresponding cached file in the top-level 'discoveries' directory at `absolutePath` with
// `permissions`, sending the intended file name to an `updateChannel` regardless of any error in
// the update.
//
// To avoid unnecessary updates due to nondeterministic JSON field ordering from live Discovery docs
// for some APIs, UpdateAPI updates only files with meaningful changes, as determined by deep
// equality of maps parsed from JSON, ignoring changes to top-level `etag` and `revision` fields.
func UpdateAPI(API apiInfo, absolutePath string, permissions os.FileMode, updateChannel chan string) error {
	fmt.Printf("Updating API: %v %v ...\n", API.Name, API.Version)
	filename := API.Name + "." + API.Version + ".json"
	updateChannel <- filename
	path := path.Join(absolutePath, filename)

	oldDisco, err := discoFromFile(path)
	if err != nil {
		return err
	}
	oldAPI, err := parseAPI(oldDisco)
	if err != nil {
		return fmt.Errorf("Error parsing Discovery doc from %v: %v", path, err)
	}

	newDisco, err := discoFromURL(API.DiscoveryRestURL)
	if err != nil {
		return err
	}
	newAPI, err := parseAPI(newDisco)
	if err != nil {
		return fmt.Errorf("Error parsing Discovery doc from %v: %v", API.DiscoveryRestURL, err)
	}

	// If "revision" is nil or not a string, the empty string is returned.
	newRevision, _ := newAPI["revision"].(string)
	oldRevision, _ := oldAPI["revision"].(string)
	// Do nothing if the revision of the new API is older than what already exists.
	if newRevision < oldRevision {
		return fmt.Errorf("Error validating Discovery doc revision from %v: %v < %v", API.DiscoveryRestURL, newRevision, oldRevision)
	}

	if oldAPI == nil || !sameAPI(oldAPI, newAPI) {
		if err := ioutil.WriteFile(path, newDisco, permissions); err != nil {
			return fmt.Errorf("Error writing Discovery doc to %v: %v", path, err)
		}
	}
	return nil
}

// discoFromFile returns the Discovery `contents` of the file at `absolutePath`, or nil if the
// file does not exist.
func discoFromFile(absolutePath string) (contents []byte, err error) {
	_, err = os.Stat(absolutePath)
	if os.IsNotExist(err) {
		err = nil
		return
	}
	contents, err = ioutil.ReadFile(absolutePath)
	if err != nil {
		err = fmt.Errorf("Error reading Discovery doc from %v: %v", absolutePath, err)
		return
	}
	return
}

// discoFromURL returns the Discovery `contents` at `URL`.
func discoFromURL(URL string) (contents []byte, err error) {
	response, err := http.Get(URL)
	// Note that err is nil for non-200 responses.
	if err != nil {
		err = fmt.Errorf("Error downloading Discovery doc from %v: %v", URL, err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		// Fail if the status code is not 200.
		// This can happen if a service is listed in the Discovery
		// directory, but the Discovery document is not accessible.
		// In this case, the existing Discovery document is left in
		// place until the directory is updated to delist the service.
		// At that point, the updatedisco script will delete the
		// Discovery document.
		err = fmt.Errorf("Error downloading Discovery doc from %v: %v response", URL, response.StatusCode)
		return
	}
	contents, err = ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("Error reading Discovery doc from %v: %v", URL, err)
		return
	}
	return
}

// parseAPI returns an `API` map comprising a nested data structure corresponding to JSON
// `discovery` data.
func parseAPI(discovery []byte) (API map[string]interface{}, err error) {
	if discovery == nil {
		return
	}
	err = json.Unmarshal(discovery, &API)
	return
}

// sameAPI returns true if maps representing `apiA` and `apiB` are deeply equal, ignoring
// differences in top-level `etag` and `revision` field values. Maps are expected to result from
// `parseAPI` and assumed non-nil.
func sameAPI(apiA, apiB map[string]interface{}) bool {
	if len(apiA) != len(apiB) {
		return false
	}
	for field, valueA := range apiA {
		valueB, inB := apiB[field]
		if !(inB && reflect.DeepEqual(valueA, valueB) || field == "etag" || field == "revision") {
			return false
		}
	}
	return true
}
