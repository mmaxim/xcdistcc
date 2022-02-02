package client

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"mmaxim.org/xcdistcc/common"
)

type RandRemoteSelector struct {
	remotes             []Remote
	preprocessorRemotes []Remote
}

func NewRandConnSelector(remotes []Remote) *RandRemoteSelector {
	var prs []Remote
	for _, remote := range remotes {
		if remote.HasPower(PreprocessorPower) {
			prs = append(prs, remote)
		}
	}
	return &RandRemoteSelector{
		remotes:             remotes,
		preprocessorRemotes: prs,
	}
}

func (c *RandRemoteSelector) getRandRemote(remotes []Remote) (res Remote, err error) {
	if len(remotes) == 0 {
		return res, errors.New("no remotes available")
	}
	remote := remotes[rand.Intn(len(remotes))]
	if !strings.Contains(remote.Address, ":") {
		remote.Address = fmt.Sprintf("%s:%d", remote.Address, common.DefaultListenPort)
	}
	return remote, nil
}

func (c *RandRemoteSelector) GetRemote() (Remote, error) {
	return c.getRandRemote(c.remotes)
}

func (c *RandRemoteSelector) GetRemoteWithPreprocessor() (Remote, error) {
	return c.getRandRemote(c.preprocessorRemotes)
}
