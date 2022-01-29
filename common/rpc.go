package common

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
)

func RPCSendRaw(conn net.Conn, raw []byte) error {
	var gzipBuf bytes.Buffer
	compressor := gzip.NewWriter(&gzipBuf)
	if _, err := io.Copy(compressor, bytes.NewBuffer(raw)); err != nil {
		return errors.Wrap(err, "failed to compress")
	}
	if err := compressor.Close(); err != nil {
		return errors.Wrap(err, "failed to close compress")
	}

	dat := gzipBuf.Bytes()
	var sz uint32
	sz = uint32(len(dat))
	errCh := make(chan error, 1)
	go func() {
		if err := binary.Write(conn, binary.BigEndian, sz); err != nil {
			errCh <- errors.Wrap(err, "failed to write len")
			return
		}
		buf := bytes.NewBuffer(dat)
		if _, err := io.CopyN(conn, buf, int64(sz)); err != nil {
			errCh <- errors.Wrap(err, "failed to write msg")
			return
		}
		errCh <- nil
	}()
	select {
	case err := <-errCh:
		return err
	case <-time.After(time.Minute):
		return errors.New("rpc send timeout")
	}
}

func RPCRecvRaw(conn net.Conn) (res []byte, err error) {
	var sz uint32
	if err := binary.Read(conn, binary.BigEndian, &sz); err != nil {
		return res, errors.Wrap(err, "failed to read response size")
	}
	resp := make([]byte, sz)
	errCh := make(chan error, 1)
	go func() {
		if _, err := io.ReadFull(conn, resp); err != nil {
			errCh <- errors.Wrap(err, "failed to read response")
			return
		}
		errCh <- nil
	}()
	select {
	case err := <-errCh:
		if err != nil {
			return res, err
		}
	case <-time.After(time.Minute):
		return res, errors.New("rpc recv timeout")
	}

	decompressor, err := gzip.NewReader(bytes.NewBuffer(resp))
	if err != nil {
		return res, errors.Wrap(err, "failed to decompress")
	}
	return io.ReadAll(decompressor)
}

func DoRPC[ReqTyp any, PayloadTyp any](conn net.Conn, method string, req ReqTyp) (res PayloadTyp, err error) {
	cmdreq := Cmd{
		Name: method,
	}
	dat, err := msgpack.Marshal(req)
	if err != nil {
		return res, errors.Wrap(err, "failed to encode req args")
	}
	cmdreq.Args = dat
	if dat, err = msgpack.Marshal(cmdreq); err != nil {
		return res, errors.Wrap(err, "failed to encode req")
	}

	// send/recv
	if err := RPCSendRaw(conn, dat); err != nil {
		return res, err
	}
	respCh := make(chan []byte, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := RPCRecvRaw(conn)
		if err != nil {
			errCh <- err
		}
		respCh <- resp
	}()
	var resp []byte
	select {
	case resp = <-respCh:
	case err := <-errCh:
		return res, err
	case <-time.After(time.Minute):
		return res, errors.New("timed out waiting for response")
	}

	// parse response
	var cmdres CmdResponse
	if err := msgpack.Unmarshal(resp, &cmdres); err != nil {
		return res, errors.Wrap(err, "failed to decode response")
	}
	if !cmdres.Success {
		return res, errors.New(*cmdres.ErrorMsg)
	}
	if err := msgpack.Unmarshal(cmdres.Payload, &res); err != nil {
		return res, errors.Wrap(err, "fsailedf to decode payload")
	}
	return res, nil
}
