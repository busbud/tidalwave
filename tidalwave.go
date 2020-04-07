// Package main is the entry point for Tidalwave.
package main

import (
	"log"

	"github.com/busbud/tidalwave/cmd"
)

func main() {
	if err := cmd.New().Execute(); err != nil {
		log.Fatal(err)
	}
}
