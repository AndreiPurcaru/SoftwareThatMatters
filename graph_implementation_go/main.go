package main

import (
	"github.com/AJMBrands/SoftwareThatMatters/cmd"
)

func main() {
	//TODO: Move to graph.go; Integrate nicely with cli
	// To use the cli: go run main.go start.
	cmd.Execute()
}
