package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/client"
	"mmaxim.org/xcdistcc/common"
)

type ConfigRemote struct {
	Address   string
	PublicKey string
	Powers    []string
}

func (r ConfigRemote) ToRemote() (res client.Remote, err error) {
	res.Address = r.Address
	if len(r.PublicKey) > 0 {
		res.PublicKey = new(common.PublicKey)
		if *res.PublicKey, err = common.NewPublicKeyFromString(r.PublicKey); err != nil {
			return res, err
		}
	}
	if len(r.Powers) > 0 {
		for _, power := range r.Powers {
			switch power {
			case "preprocess":
				res.Powers = append(res.Powers, client.PreprocessorPower)
			case "compile":
				res.Powers = append(res.Powers, client.CompilePower)
			}
		}
	} else {
		res.Powers = []client.Power{client.CompilePower}
	}
	return res, nil
}

type ConfigFile struct {
	Remotes []ConfigRemote
}

type Config struct {
	Remotes        []client.Remote
	Logger         common.Logger
	RemoteSelector client.RemoteSelector
	Preprocessor   client.Preprocessor
}

func LoadConfig(path string) (config *Config, err error) {
	config = new(Config)
	dat, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}
	var configFile ConfigFile
	if err := json.Unmarshal(dat, &configFile); err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	config.Remotes = make([]client.Remote, len(configFile.Remotes))
	for index, remote := range configFile.Remotes {
		// add default port to remotes missing it
		if !strings.Contains(remote.Address, ":") {
			remote.Address = fmt.Sprintf("%s:%d", remote.Address, common.DefaultListenPort)
		}
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
