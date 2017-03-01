// Package main provides a standalone executable `updatedisco` to update the local Discovery doc
// cache from the live Discovery service in a top-level directory 'discoveries', which must
// exist. Run anywhere in the `discovery-artifact-manager` repository.
package main

import (
	"fmt"

	"discovery-artifact-manager/main/common"
)

func main() {
	if err := common.UpdateDiscos(); err != nil {
		fmt.Println("Error updating some APIs:")
		fmt.Println(err)
	}
}
