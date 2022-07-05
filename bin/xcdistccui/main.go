package main

import (
	"log"
	"os"

	"fyne.io/fyne/v2/app"
	"mmaxim.org/xcdistcc/bin"
	"mmaxim.org/xcdistcc/client"
	"mmaxim.org/xcdistcc/ui"
)

func main() {
	configFile, err := bin.LoadConfigFile()
	if err != nil {
		log.Printf("failed to load config file: %s", err)
		os.Exit(3)
	}

	remotes := make([]client.Remote, len(configFile.Remotes))
	for index, remote := range configFile.Remotes {
		if remotes[index], err = remote.ToRemote(); err != nil {
			log.Printf("invalid remote: %s", err)
			os.Exit(3)
		}
	}

	a := app.New()
	ui.NewMainWindow(a, remotes).ShowAndRun()
}
