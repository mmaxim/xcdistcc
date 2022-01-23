package common

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"

	"github.com/pkg/errors"
)

func RPCSendRaw(conn net.Conn, dat []byte) error {
	var sz uint32
	sz = uint32(len(dat))
	if err := binary.Write(conn, binary.BigEndian, sz); err != nil {
		return errors.Wrap(err, "failed to write len")
	}
	buf := bytes.NewBuffer(dat)
	if _, err := io.Copy(conn, buf); err != nil {
		return errors.Wrap(err, "failed to write msg")
	}
	return nil
}

func RPCRecvRaw(conn net.Conn) (res []byte, err error) {
	var sz uint32
	if err := binary.Read(conn, binary.BigEndian, &sz); err != nil {
		return res, errors.Wrap(err, "failed to read response size")
	}
	resp := make([]byte, sz)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return res, errors.Wrap(err, "failed to read response")
	}
	return resp, nil
}

func DoRPC[ReqTyp any, PayloadTyp any](conn net.Conn, method string, req ReqTyp) (res PayloadTyp, err error) {
	cmdreq := Cmd{
		Name: method,
	}
	dat, err := json.Marshal(req)
	if err != nil {
		return res, errors.Wrap(err, "failed to encode req args")
	}
	cmdreq.Args = dat
	if dat, err = json.Marshal(cmdreq); err != nil {
		return res, errors.Wrap(err, "failed to encode req")
	}

	// send/recv
	if err := RPCSendRaw(conn, dat); err != nil {
		return res, err
	}
	resp, err := RPCRecvRaw(conn)
	if err != nil {
		return res, err
	}

	// parse response
	var cmdres CmdResponse
	if err := json.Unmarshal(resp, &cmdres); err != nil {
		return res, errors.Wrap(err, "failed to decode response")
	}
	if !cmdres.Success {
		return res, errors.New(*cmdres.ErrorMsg)
	}
	if err := json.Unmarshal(cmdres.Payload, &res); err != nil {
		return res, errors.Wrap(err, "fsailedf to decode payload")
	}
	return res, nil
}
