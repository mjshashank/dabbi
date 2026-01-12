package main

import (
	"github.com/mjshashank/dabbi/internal/cli"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cli.SetVersion(Version, BuildTime)
	cli.Execute()
}
