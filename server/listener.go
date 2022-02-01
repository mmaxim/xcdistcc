package server

import (
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/crypto/nacl/box"
	"mmaxim.org/xcdistcc/common"
)

type Listener struct {
	*common.LabelLogger
	runner     *Runner
	address    string
	keyPair    *common.KeyPair
	listener   net.Listener
	shutdownCh chan struct{}
}

func NewListener(runner *Runner, address string, keyPair *common.KeyPair, logger common.Logger) *Listener {
	return &Listener{
		LabelLogger: common.NewLabelLogger("Listener", logger),
		runner:      runner,
		address:     address,
		keyPair:     keyPair,
		shutdownCh:  make(chan struct{}),
	}
}

func (r *Listener) Run() (err error) {
	go r.signalHandler()
	if r.listener, err = net.Listen("tcp", r.address); err != nil {
		r.Debug("Run: failed to listen on address: %s", err)
		return err
	}
	defer r.listener.Close()
	r.Debug("listening on: %s", r.address)
	if r.keyPair != nil {
		r.Debug("secure connection: public key: %s", r.keyPair.Public)
	}
	for {
		connCh := make(chan net.Conn)
		errCh := make(chan error)
		go r.acceptConn(connCh, errCh)
		select {
		case <-r.shutdownCh:
			return nil
		case conn := <-connCh:
			go r.serve(conn)
		case err := <-errCh:
			log.Printf("failed to accept connection: %s", err)
		}
	}
}

func (r *Listener) acceptConn(connCh chan net.Conn, errCh chan error) {
	conn, err := r.listener.Accept()
	if err != nil {
		errCh <- err
	}
	connCh <- conn
}

func (r *Listener) signalHandler() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	close(r.shutdownCh)
}

func (r *Listener) sendResponse(payload interface{}, err error, conn net.Conn, secret *common.SharedSecret) error {
	var response common.CmdResponse
	if err != nil {
		response.Success = false
		response.ErrorMsg = new(string)
		*response.ErrorMsg = err.Error()
	} else {
		response.Success = true
		dat, err := msgpack.Marshal(payload)
		if err != nil {
			r.Debug("sendResponse: failed to marshal payload: %s", err)
			return err
		}
		response.Payload = dat
	}
	dat, err := msgpack.Marshal(response)
	if err != nil {
		r.Debug("sendResponse: failed to marshal response: %s", err)
		return err
	}
	if err := common.RPCSendRaw(conn, dat, secret); err != nil {
		r.Debug("sendResponse: failed to send response: %s", err)
		return err
	}
	return nil
}

func (r *Listener) handleCommand(cmd common.Cmd, conn net.Conn, secret *common.SharedSecret) error {
	switch cmd.Name {
	case common.MethodCompile:
		var compile common.CompileCmd
		if err := msgpack.Unmarshal(cmd.Args, &compile); err != nil {
			r.Debug("handleCommand: failed to parse compile args: %s", err)
			return err
		}
		payload, err := r.runner.Compile(compile, "")
		return r.sendResponse(payload, err, conn, secret)
	case common.MethodStatus:
		return r.sendResponse(r.runner.Status(), nil, conn, secret)
	default:
		r.Debug("handleCommand: unknown command: %s", cmd.Name)
		return errors.New("unknown command")
	}
}

func (r *Listener) handshake(conn net.Conn) (res *common.SharedSecret, err error) {
	var out [32]byte
	if _, err := io.ReadFull(conn, out[:]); err != nil {
		return nil, err
	}
	pk := common.NewPublicKey(out)
	box.Precompute(&out, pk.RawPtr(), r.keyPair.Private.RawPtr())
	secret := common.NewSharedSecret(out)
	return &secret, nil
}

func (r *Listener) serve(conn net.Conn) {
	defer conn.Close()
	var err error
	var sharedSecret *common.SharedSecret
	if r.keyPair != nil {
		if sharedSecret, err = r.handshake(conn); err != nil {
			r.Debug("serve: failed handshake: %s", err)
			return
		}
	}
	for {
		dat, err := common.RPCRecvRaw(conn, sharedSecret)
		if err != nil {
			if errors.Unwrap(err) == io.EOF {
				r.Debug("serve: failed to recv: %s", err)
			}
			return
		}
		var cmd common.Cmd
		if err := msgpack.Unmarshal(dat, &cmd); err != nil {
			r.Debug("serve: invalid msgpack: %s", err)
			return
		}
		if err := r.handleCommand(cmd, conn, sharedSecret); err != nil {
			r.Debug("serve: failed to handle command: %s", err)
			return
		}
	}
}
