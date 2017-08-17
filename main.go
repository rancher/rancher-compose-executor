package main

import (
	"os"
	"path"

	"github.com/rancher/rancher-compose-executor/executor"
	"github.com/rancher/rancher-compose-executor/testcli"
)

func main() {
	if path.Base(os.Args[0]) == "rancher-compose-executor" {
		executor.Main()
	} else {
		testcli.Main()
	}
}
