package client

import (
	"net"

	"mmaxim.org/xcdistcc/common"
)

type Power int

const (
	PreprocessorPower Power = iota
	CompilePower
)

type Remote struct {
	Address   string
	PublicKey *common.PublicKey
	Powers    []Power
}

func (r Remote) HasPower(target Power) bool {
	for _, power := range r.Powers {
		if power == target {
			return true
		}
	}
	return false
}

type RemoteConn struct {
	Conn   net.Conn
	Secret *common.SharedSecret
}

func NewRemoteConn(conn net.Conn, secret *common.SharedSecret) *RemoteConn {
	return &RemoteConn{
		Conn:   conn,
		Secret: secret,
	}
}

func DialRemote(remote Remote) (*RemoteConn, error) {
	if remote.PublicKey == nil {
		conn, err := net.Dial("tcp", remote.Address)
		if err != nil {
			return nil, err
		}
		return NewRemoteConn(conn, nil), err
	}
	conn, secret, err := common.DialEncrypted(remote.Address, *remote.PublicKey)
	if err != nil {
		return nil, err
	}
	return NewRemoteConn(conn, secret), nil
}
