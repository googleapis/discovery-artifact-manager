package common

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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
	if err := os.RemoveAll(discoPath); err != nil {
		return fmt.Errorf("Error removing old Discovery docs: %v", err)
	}
	perm := info.Mode()
	if err := os.MkdirAll(discoPath, perm); err != nil {
		return fmt.Errorf("Error re-initializing path for Discovery docs: %v", err)
	}

	fmt.Println(fmt.Sprintf("Fetching Discovery doc index from %v ...", discoURL))
	response, err := http.Get(discoURL)
	if err != nil {
		return fmt.Errorf("Error fetching Discovery doc index: %v", err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("Error reading Discovery doc index: %v", err)
	}

	fmt.Println("Reading Discovery doc index ...")
	var index struct {
		Items []apiInfo
	}
	if err := json.Unmarshal(body, &index); err != nil {
		return fmt.Errorf("Error parsing Discovery doc index: %v", err)
	}

	fmt.Println(fmt.Sprintf("Updating local Discovery docs in %v:", discoPath))
	// Make Discovery doc file permissions like parent directory (no execute)
	perm &= 0666
	var update sync.WaitGroup
	errc := make(chan error, len(index.Items))
	for _, api := range index.Items {
		update.Add(1)
		go func(api apiInfo) {
			defer update.Done()
			if err := UpdateAPI(api, discoPath, perm); err != nil {
				fmt.Println(err)
				errc <- fmt.Errorf("Error updating %v %v: %v", api.Name, api.Version, err)
			}
		}(api)
	}
	update.Wait()
	close(errc)
	var errs errorlist.Errors
	for err := range errc {
		errs.Add(err)
	}
	return errs.Error()
}

// UpdateAPI updates the local Discovery doc file for an API indexed by the live Discovery service.
func UpdateAPI(api apiInfo, discoPath string, perm os.FileMode) error {
	fmt.Println(fmt.Sprintf("Updating API: %v %v ...", api.Name, api.Version))
	response, err := http.Get(api.DiscoveryRestUrl)
	if err != nil {
		return fmt.Errorf("Error downloading Discovery doc from %v: %v", api.DiscoveryRestUrl, err)
	}
	defer response.Body.Close()

	filename := path.Join(discoPath, api.Name+"."+api.Version+".json")
	disco, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return fmt.Errorf("Error creating local discovery doc file: %v", filename)
	}
	defer disco.Close()

	if _, err := io.Copy(disco, response.Body); err != nil {
		return fmt.Errorf("Error writing local Discovery doc file: %v", filename)
	}
	return nil
}
