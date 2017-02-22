package main

import (
	"fmt"

	"discovery-artifact-manager/main/common"
)

func main() {
	if err := common.UpdateDiscos(); err != nil {
		fmt.Println(err)
	}
}
