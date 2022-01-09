package main

import (
	"os"
	"strings"

	"mmaxim.org/xcdistcc/client"
)

func main() {
	dispatcher := client.NewDispatcher(nil)
	if err := dispatcher.Run(strings.Join(os.Args[1:], " ")); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}
