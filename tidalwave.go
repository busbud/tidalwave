package main

import (
	"log"

	"github.com/dustinblackman/tidalwave/cmd"
)

func main() {
	if err := cmd.New().Execute(); err != nil {
		log.Fatal(err)
	}
}
