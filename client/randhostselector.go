package client

import (
	"fmt"
	"math/rand"
	"strings"

	"mmaxim.org/xcdistcc/common"
)

type RandConnSelector struct {
	hosts []string
}

func NewRandConnSelector(hosts []string) *RandConnSelector {
	return &RandConnSelector{
		hosts: hosts,
	}
}

func (c *RandConnSelector) GetConn() (string, error) {
	host := c.hosts[rand.Intn(len(c.hosts))]
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, common.DefaultListenPort)
	}
	return host, nil
}
