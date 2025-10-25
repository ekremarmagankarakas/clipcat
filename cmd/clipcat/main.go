package main

import (
	"clipcat/pkg/clipcat"
	"fmt"
	"os"
)

func main() {
	cfg := clipcat.ParseArgs()

	if err := clipcat.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}