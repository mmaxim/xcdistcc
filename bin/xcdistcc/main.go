package main

import (
	"log"
	"os"
	"strings"

	"mmaxim.org/xcdistcc/client"
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Printf("failed to load config: %s", err)
		os.Exit(3)
	}

	dispatcher := client.NewDispatcher(config.RemoteSelector, config.Preprocessor, config.Logger)
	if err := dispatcher.Run(strings.Join(os.Args[1:], " ")); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}
