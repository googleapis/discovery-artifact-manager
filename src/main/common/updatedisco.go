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

type files map[string]bool

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
	olds, err := ioutil.ReadDir(discoPath)
	if err != nil {
		return fmt.Errorf("Error reading path for Discovery docs: %v", discoPath)
	}
	old := make(map[string]bool, len(olds))
	for _, file := range olds {
		old[file.Name()] = true
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

	fmt.Printf("Updating local Discovery docs in %v:\n", discoPath)
	// Make Discovery doc file permissions like parent directory (no execute)
	perm := info.Mode() & 0666

	var collect sync.WaitGroup
	var errs errorlist.Errors
	errc := make(chan error, len(index.Items))
	collect.Add(1)
	go func() {
		defer collect.Done()
		for err := range errc {
			fmt.Println(err)
			errs.Add(err)
		}
	}()

	newc := make(chan string, len(index.Items))
	collect.Add(1)
	go func() {
		defer collect.Done()
		for new := range newc {
			delete(old, new)
		}
	}()

	var update sync.WaitGroup
	for _, api := range index.Items {
		update.Add(1)
		go func(api apiInfo) {
			defer update.Done()
			if err := UpdateAPI(api, discoPath, perm, newc); err != nil {
				errc <- fmt.Errorf("Error updating %v %v: %v", api.Name, api.Version, err)
			}
		}(api)
	}
	update.Wait()
	close(errc)
	close(newc)
	collect.Wait()
	for filename := range old {
		filepath := path.Join(discoPath, filename)
		if err := os.Remove(filepath); err != nil {
			errs.Add(fmt.Errorf("Error deleting expired Discovery doc %v: %v", filepath, err))
		}
	}
	return errs.Error()
}

// UpdateAPI updates the local Discovery doc file for an API indexed by the live Discovery service.
func UpdateAPI(api apiInfo, discoPath string, perm os.FileMode, newc chan string) error {
	filename := api.Name + "." + api.Version + ".json"
	newc <- filename
	filepath := path.Join(discoPath, filename)

	disco, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return fmt.Errorf("Error creating local discovery doc file: %v", filepath)
	}
	defer disco.Close()

	fmt.Printf("Updating API: %v %v ...\n", api.Name, api.Version)
	response, err := http.Get(api.DiscoveryRestUrl)
	if err != nil {
		return fmt.Errorf("Error downloading Discovery doc from %v: %v", api.DiscoveryRestUrl, err)
	}
	defer response.Body.Close()

	if _, err := io.Copy(disco, response.Body); err != nil {
		return fmt.Errorf("Error writing local Discovery doc file: %v", filepath)
	}
	return nil
}
