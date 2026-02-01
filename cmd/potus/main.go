package main

import (
	"fmt"
	"os"

	"github.com/taaha3244/potus/internal/cli"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	if err := cli.Execute(version, commit, date); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
