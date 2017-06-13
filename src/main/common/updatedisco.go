package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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
	absolutePath, dir, files, err := readDiscoCache()
	if err != nil {
		return
	}

	index, err := readDiscoIndex()
	if err != nil {
		return
	}

	updated, errs := writeDiscoCache(index, absolutePath, dir)

	cleanDiscoCache(absolutePath, files, updated, &errs)
	err = errs.Error()
	if err != nil {
		return
	}

	names = make([]string, 0, len(updated))
	for filename, _ := range updated {
		filepath := path.Join(absolutePath, filename)
		names = append(names, filepath)
	}
	return
}

// readDiscoCache returns the `absolutePath` to the top-level 'discoveries' directory along with its
// `directory` attributes and those of all discovery `files` therein.
func readDiscoCache() (absolutePath string, directory os.FileInfo, files []os.FileInfo, err error) {
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
	files, err = ioutil.ReadDir(absolutePath)
	if err != nil {
		err = fmt.Errorf("Error reading path for Discovery docs: %v", absolutePath)
	}
	return
}

// readDiscoIndex returns an `index` of API attributes extracted from the JSON index returned by the
// live Discovery service.
func readDiscoIndex() (index *apiIndex, err error) {
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
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("Error reading Discovery doc index: %v", err)
		return
	}

	fmt.Println("Parsing Discovery doc index ...")
	index = &apiIndex{}
	err = json.Unmarshal(body, index)
	if err != nil {
		err = fmt.Errorf("Error parsing Discovery doc index: %v", err)
		return
	}
	return
}

// writeDiscoCache updates (creates or replaces) Discovery doc files in the top-level 'discoveries'
// directory (given its `absolutePath` and `directory` attributes) as needed to update descriptions
// of APIs in the `index` (assumed not to contain duplicates). It returns a map containing `updated`
// file basenames corresponding to live APIs, and accumulates `errors` from all updates.
func writeDiscoCache(index *apiIndex, absolutePath string, directory os.FileInfo) (updated map[string]bool, errors errorlist.Errors) {
	fmt.Printf("Updating local Discovery docs in %v:\n", absolutePath)
	size := len(index.Items)
	// Make Discovery doc file permissions like parent directory (no execute)
	perm := directory.Mode() & 0666

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

// cleanDiscoCache deletes those `files` in the top-level 'discoveries' directory at `absolutePath`
// whose names do not appear in the map of `updated` files, and accumulates any further `errors`.
func cleanDiscoCache(absolutePath string, files []os.FileInfo, updated map[string]bool, errors *errorlist.Errors) {
	for _, file := range files {
		filename := file.Name()
		if !updated[filename] {
			filepath := path.Join(absolutePath, filename)
			if err := os.Remove(filepath); err != nil {
				errors.Add(fmt.Errorf("Error deleting expired Discovery doc %v: %v", filepath, err))
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
	filepath := path.Join(absolutePath, filename)

	oldDisco, err := discoFromFile(filepath)
	if err != nil {
		return err
	}
	oldAPI, err := parseAPI(oldDisco)
	if err != nil {
		return fmt.Errorf("Error parsing Discovery doc from %v: %v", filepath, err)
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
		if err := ioutil.WriteFile(filepath, newDisco, permissions); err != nil {
			return fmt.Errorf("Error writing Discovery doc to %v: %v", filepath, err)
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
