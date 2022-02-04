package main

import (
	"os"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/bin"
	"mmaxim.org/xcdistcc/client"
	"mmaxim.org/xcdistcc/common"
)

type Config struct {
	Remotes        []client.Remote
	Logger         common.Logger
	RemoteSelector client.RemoteSelector
	Preprocessor   client.Preprocessor
}

func LoadConfig() (config *Config, err error) {
	config = new(Config)
	configFile, err := bin.LoadConfigFile()
	if err != nil {
		return nil, err
	}

	config.Remotes = make([]client.Remote, len(configFile.Remotes))
	for index, remote := range configFile.Remotes {
		if config.Remotes[index], err = remote.ToRemote(); err != nil {
			return nil, errors.Wrap(err, "invalid remote")
		}
	}

	// read environment variables for other config
	verbose := len(os.Getenv("XCDISTCC_VERBOSE")) > 0
	logPath := os.Getenv("XCDISTCC_LOGPATH")
	if verbose {
		if len(logPath) > 0 {
			config.Logger, err = common.NewStdLoggerWithFilepath(logPath)
			if err != nil {
				return nil, errors.Wrap(err, "failed to open log file path")
			}
		} else {
			config.Logger = common.NewStdLogger()
		}
	} else {
		config.Logger = common.NewQuietLogger()
	}

	remoteSelectorStr := os.Getenv("XCDISTCC_REMOTESELECTOR")
	switch remoteSelectorStr {
	case "random":
		config.RemoteSelector = client.NewRandConnSelector(config.Remotes)
	case "queuesize":
		fallthrough
	default:
		config.RemoteSelector = client.NewStatusRemoteSelector(config.Remotes, config.Logger)
	}

	preprocessorStr := os.Getenv("XCDISTCC_PREPROCESSOR")
	switch preprocessorStr {
	case "includefinder":
		config.Preprocessor = client.NewIncludeFinder(config.Logger)
	case "remote":
		config.Preprocessor = client.NewRemotePreprocessor(config.RemoteSelector,
			client.NewClangPreprocessor(config.Logger), config.Logger)
	case "local":
		fallthrough
	default:
		config.Preprocessor = client.NewClangPreprocessor(config.Logger)
	}

	return config, nil
}
