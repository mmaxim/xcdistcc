package client

import (
	"net"

	"mmaxim.org/xcdistcc/common"
)

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
	if len(remote.PublicKeyStr) == 0 {
		conn, err := net.Dial("tcp", remote.Address)
		if err != nil {
			return nil, err
		}
		return NewRemoteConn(conn, nil), err
	}
	pk, err := common.NewPublicKeyFromString(remote.PublicKeyStr)
	if err != nil {
		return nil, err
	}
	conn, secret, err := common.DialEncrypted(remote.Address, pk)
	if err != nil {
		return nil, err
	}
	return NewRemoteConn(conn, secret), nil
}
