package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"discovery-artifact-manager/common/environment"
)

var discoURL = "https://www.googleapis.com/discovery/v1/apis"
var discoPath string

type API struct {
	Name, Version, DiscoveryRestUrl string
}

// UpdateDiscos updates local discovery doc files for all APIs indexed by live discovery service
func UpdateDiscos() error {
	root, err := environment.RepoRoot()
	if err != nil {
		return fmt.Errorf("Error finding path for discovery docs: %v", err)
	}
	discoPath = path.Join(root, "discoveries")
	info, err := os.Stat(discoPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("Error finding path for discovery docs: %v", discoPath)
	}
	if err := os.RemoveAll(discoPath); err != nil {
		return fmt.Errorf("Error removing old discovery docs: %v", err)
	}
	if err := os.MkdirAll(discoPath, info.Mode()); err != nil {
		return fmt.Errorf("Error re-initializing path for discovery docs: %v", err)
	}
	fmt.Println(fmt.Sprintf("Fetching discovery doc index from %v ...", discoURL))
	response, err := http.Get(discoURL)
	if err != nil {
		return fmt.Errorf("Error fetching discovery doc index: %v", err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("Error reading discovery doc index: %v", err)
	}
	type Index struct {
		Items []API
	}
	// var index Index
	index := Index{}
	if err := json.Unmarshal(body, &index); err != nil {
		return fmt.Errorf("Error parsing discovery doc index: %v", err)
	}
	var failed []API
	fmt.Println(fmt.Sprintf("Updating local discovery docs in %v:", discoPath))
	for _, api := range index.Items {
		if err := UpdateAPI(api); err != nil {
			failed = append(failed, api)
			fmt.Println(err)
		}
	}
	if len(failed) > 0 {
		return errors.New(fmt.Sprintf("Error updating some APIs: %v", failed))
	}
	return nil
}

// UpdateAPI updates local discovery doc file for an API indexed by live discovery service
func UpdateAPI(api API) error {
	fmt.Println(fmt.Sprintf("Updating API: %v %v ...", api.Name, api.Version))
	response, err := http.Get(api.DiscoveryRestUrl)
	if err != nil {
		return fmt.Errorf("Error downloading discovery doc from %v: %v", api.DiscoveryRestUrl, err)
	}
	defer response.Body.Close()
	filename := path.Join(discoPath, api.Name+"."+api.Version+".json")
	disco, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Error creating local discovery doc file: %v", filename)
	}
	defer disco.Close()
	if _, err := io.Copy(disco, response.Body); err != nil {
		return fmt.Errorf("Error writing local discovery doc file: %v", filename)
	}
	return nil
}
