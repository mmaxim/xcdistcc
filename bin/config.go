package bin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

func LoadConfigFile() (*ConfigFile, error) {
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

	dat, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}
	var configFile ConfigFile
	if err := json.Unmarshal(dat, &configFile); err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	for index, remote := range configFile.Remotes {
		if !strings.Contains(remote.Address, ":") {
			remote.Address = fmt.Sprintf("%s:%d", remote.Address, common.DefaultListenPort)
		}
		configFile.Remotes[index] = remote
	}

	return &configFile, nil
}
