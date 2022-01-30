package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"mmaxim.org/xcdistcc/client"
)

func main() {
	var path string
	envStr := os.Getenv("XCDISTCC_CONFIGFILE")
	if len(envStr) != 0 {
		path = envStr
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("failed to get user home directory: %s", err)
			os.Exit(3)
		}
		path = filepath.Join(filepath.Join(homeDir, ".xcdistcc"), "config.json")
	}
	config, err := LoadConfig(path)
	if err != nil {
		log.Printf("failed to load config: %s", err)
		os.Exit(3)
	}

	dispatcher := client.NewDispatcher(client.NewStatusHostSelector(config.Remotes, config.Logger),
		client.NewClangPreprocessor(config.Logger), config.Logger)
	if err := dispatcher.Run(strings.Join(os.Args[1:], " ")); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}
