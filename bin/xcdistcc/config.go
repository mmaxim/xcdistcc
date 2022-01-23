package main

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

type Config struct {
	Remotes []string
}

func LoadConfig(path string) (*Config, error) {
	dat, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}
	var config Config
	if err := json.Unmarshal(dat, &config); err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}
	return &config, nil
}
