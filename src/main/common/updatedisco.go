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

// UpdateDiscos updates local Discovery doc files for all APIs indexed by the live Discovery
// service, in a top-level directory 'discoveries', which must exist.
func UpdateDiscos() error {
	root, err := environment.RepoRoot()
	if err != nil {
		return fmt.Errorf("Error finding repository root directory: %v", err)
	}
	discoPath := path.Join(root, "discoveries")
	info, err := os.Stat(discoPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("Error finding path for Discovery docs: %v", discoPath)
	}
	oldFiles, err := ioutil.ReadDir(discoPath)
	if err != nil {
		return fmt.Errorf("Error reading path for Discovery docs: %v", discoPath)
	}

	fmt.Printf("Fetching Discovery doc index from %v ...\n", discoURL)
	response, err := http.Get(discoURL)
	if err != nil {
		return fmt.Errorf("Error fetching Discovery doc index: %v", err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("Error reading Discovery doc index: %v", err)
	}

	fmt.Println("Parsing Discovery doc index ...")
	var index struct {
		Items []apiInfo
	}
	if err := json.Unmarshal(body, &index); err != nil {
		return fmt.Errorf("Error parsing Discovery doc index: %v", err)
	}
	size := len(index.Items)

	fmt.Printf("Updating local Discovery docs in %v:\n", discoPath)
	// Make Discovery doc file permissions like parent directory (no execute)
	perm := info.Mode() & 0666

	var collect sync.WaitGroup
	var errs errorlist.Errors
	errChan := make(chan error, size)
	collect.Add(1)
	go func() {
		defer collect.Done()
		for err := range errChan {
			fmt.Println(err)
			errs.Add(err)
		}
	}()

	updated := make(map[string]bool, size)
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
			if err := UpdateAPI(api, discoPath, perm, updateChan); err != nil {
				errChan <- fmt.Errorf("Error updating %v %v: %v", api.Name, api.Version, err)
			}
		}(api)
	}
	update.Wait()
	close(errChan)
	close(updateChan)
	collect.Wait()
	for _, file := range oldFiles {
		filename := file.Name()
		if !updated[filename] {
			filepath := path.Join(discoPath, filename)
			if err := os.Remove(filepath); err != nil {
				errs.Add(fmt.Errorf("Error deleting expired Discovery doc %v: %v", filepath, err))
			}
		}
	}
	return errs.Error()
}

// UpdateAPI updates the local Discovery doc file for an API indexed by the live Discovery service,
// sending the intended file name to a channel regardless of any error in the update.
//
// To avoid unnecessary updates due to nondeterministic JSON field ordering from live Discovery docs
// for some APIs, UpdateAPI updates only files with meaningful changes, as determined by deep
// equality of maps parsed from JSON, ignoring changes to top-level `etag` and `revision` fields.
func UpdateAPI(api apiInfo, discoPath string, perm os.FileMode, updateChan chan string) error {
	fmt.Printf("Updating API: %v %v ...\n", api.Name, api.Version)
	filename := api.Name + "." + api.Version + ".json"
	updateChan <- filename
	filepath := path.Join(discoPath, filename)

	oldDisco, err := discoFromFile(filepath)
	if err != nil {
		return err
	}
	oldAPI, err := parseAPI(oldDisco)
	if err != nil {
		return fmt.Errorf("Error parsing Discovery doc from %v: %v", filepath, err)
	}

	newDisco, err := discoFromURL(api.DiscoveryRestURL)
	if err != nil {
		return err
	}
	newAPI, err := parseAPI(newDisco)
	if err != nil {
		return fmt.Errorf("Error parsing Discovery doc from %v: %v", api.DiscoveryRestURL, err)
	}

	if oldAPI == nil || !sameAPI(oldAPI, newAPI) {
		if err := ioutil.WriteFile(filepath, newDisco, perm); err != nil {
			return fmt.Errorf("Error writing Discovery doc to %v: %v", filepath, err)
		}
	}
	return nil
}

func discoFromFile(filepath string) ([]byte, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, nil
	}
	disco, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("Error reading Discovery doc from %v: %v", filepath, err)
	}
	return disco, nil
}

func discoFromURL(discoURL string) ([]byte, error) {
	response, err := http.Get(discoURL)
	if err != nil {
		return nil, fmt.Errorf("Error downloading Discovery doc from %v: %v", discoURL, err)
	}
	defer response.Body.Close()
	disco, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading Discovery doc from %v: %v", discoURL, err)
	}
	return disco, nil
}

func parseAPI(disco []byte) (map[string]interface{}, error) {
	if disco == nil {
		return nil, nil
	}
	var api map[string]interface{}
	if err := json.Unmarshal(disco, &api); err != nil {
		return nil, err
	}
	return api, nil
}

// sameAPI returns true if the given maps, representing APIs parsed from Discovery docs (and assumed
// non-nil), are deeply equal, ignoring differences in top-level `etag` and `revision` fields.
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
