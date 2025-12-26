package main

import (
	"os"

	"github.com/jingkaihe/icloud-cli/cmd/icloud"
)

func main() {
	if err := icloud.Execute(); err != nil {
		os.Exit(1)
	}
}
