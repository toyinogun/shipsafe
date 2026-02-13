// Package main is the entrypoint for the ShipSafe CLI.
// It delegates all command handling to the cmd package.
package main

import (
	"os"

	"github.com/toyinlola/shipsafe/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
