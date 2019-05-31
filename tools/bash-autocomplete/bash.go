package main

import (
	"io/ioutil"
	"os"

	"github.com/busbud/tidalwave/cmd"
)

func main() {
	rootCmd := cmd.New()
	rootCmd.GenBashCompletionFile("out.sh")
	completeFile, _ := ioutil.ReadFile("out.sh")
	os.Remove("out.sh")
	os.Remove("./cmd/autocomplete.go")
	completeFileGo := `package cmd

var bashAutocomplete = ` + "`" + string(completeFile) + "\n`"
	ioutil.WriteFile("./cmd/autocomplete.go", []byte(completeFileGo), 0755)
}
