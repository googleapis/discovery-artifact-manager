package common

import (
	"encoding/json"
	"fmt"
	"io"
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
	Name, Version, DiscoveryRestUrl string
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
	previous, err := ioutil.ReadDir(discoPath)
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
	for _, file := range previous {
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

	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, perm)
	if err != nil {
		return fmt.Errorf("Error opening local Discovery doc file: %v", filepath)
	}
	defer file.Close()
	var oldDisco map[string]interface{}
	// With no existing local Discovery doc file, oldDisco remains nil
	if err := json.NewDecoder(file).Decode(&oldDisco); err != nil && err != io.EOF {
		return fmt.Errorf("Error parsing existing Discovery doc file: %v", filepath)
	}

	response, err := http.Get(api.DiscoveryRestUrl)
	if err != nil {
		return fmt.Errorf("Error downloading Discovery doc from %v: %v", api.DiscoveryRestUrl, err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("Error reading Discovery doc from %v: %v", api.DiscoveryRestUrl, err)
	}
	var newDisco map[string]interface{}
	if err := json.Unmarshal(body, &newDisco); err != nil {
		return fmt.Errorf("Error parsing Discovery doc from %v: %v", api.DiscoveryRestUrl, err)
	}

	if oldDisco == nil || !sameAPI(oldDisco, newDisco) {
		if err := file.Truncate(0); err != nil {
			return fmt.Errorf("Error erasing local Discovery doc file: %v", filepath)
		}
		if _, err := file.Seek(0, 0); err != nil {
			return fmt.Errorf("Error initializing local Discovery doc file: %v", filepath)
		}
		if _, err := file.Write(body); err != nil {
			return fmt.Errorf("Error writing local Discovery doc file: %v", filepath)
		}
	}
	return nil
}

func sameAPI(discoA, discoB map[string]interface{}) bool {
	if len(discoA) != len(discoB) {
		return false
	}
	for field, valueA := range discoA {
		if field == "etag" || field == "revision" {
			continue
		}
		valueB, inB := discoB[field]
		if !inB || !reflect.DeepEqual(valueA, valueB) {
			return false
		}
	}
	return true
}
