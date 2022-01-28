package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/common"
)

type Config struct {
	Remotes []string
	Logger  common.Logger
}

func LoadConfig(path string) (*Config, error) {
	var err error
	dat, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}
	var config Config
	if err := json.Unmarshal(dat, &config); err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	// add default port to hosts missing it
	for index, host := range config.Remotes {
		if !strings.Contains(host, ":") {
			host = fmt.Sprintf("%s:%d", host, common.DefaultListenPort)
			config.Remotes[index] = host
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
	return &config, nil
}
