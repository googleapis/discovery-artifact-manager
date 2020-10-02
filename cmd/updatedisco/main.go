// Package main provides a standalone executable `updatedisco` to update the local Discovery doc
// cache from the live Discovery service in a top-level directory 'discoveries', which must
// exist. Run anywhere in the `discovery-artifact-manager` repository. NOTE: all files in
// 'discoveries' not indexed by live Discovery service will be removed.
package main

import (
	"fmt"

	"github.com/googleapis/discovery-artifact-manager/cmd/common"
)

func main() {
	if _, err := common.UpdateDiscos(); err != nil {
		fmt.Println("Error updating APIs:")
		fmt.Println(err)
	}
}
