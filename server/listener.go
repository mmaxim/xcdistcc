package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"mmaxim.org/xcdistcc/common"
)

type Listener struct {
	*common.LabelLogger
	runner     *Runner
	address    string
	listener   net.Listener
	shutdownCh chan struct{}
}

func NewListener(runner *Runner, address string) *Listener {
	return &Listener{
		LabelLogger: common.NewLabelLogger("Listener"),
		runner:      runner,
		address:     address,
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

func (r *Listener) sendResponse(payload interface{}, err error, conn net.Conn) {
	var response common.CmdResponse
	if err != nil {
		response.Success = false
		response.ErrorMsg = new(string)
		*response.ErrorMsg = err.Error()
	} else {
		response.Success = true
		dat, err := json.Marshal(payload)
		if err != nil {
			r.Debug("sendResponse: failed to marshal payload: %s", err)
			return
		}
		response.Payload = dat
	}
	dat, err := json.Marshal(response)
	if err != nil {
		r.Debug("sendResponse: failed to marshal response: %s", err)
		return
	}
	if _, err := io.Copy(conn, bytes.NewBuffer(append(dat, common.NewLineBytes...))); err != nil {
		r.Debug("sendResponse: failed to send response: %s", err)
		return
	}
}

func (r *Listener) handleCommand(cmd common.Cmd, conn net.Conn) {
	switch cmd.Name {
	case common.MethodCompile:
		var compile common.CompileCmd
		if err := json.Unmarshal(cmd.Args, &compile); err != nil {
			r.Debug("handleCommand: failed to parse compile args: %s", err)
			return
		}
		payload, err := r.runner.Compile(compile, "")
		r.sendResponse(payload, err, conn)
	case common.MethodStatus:
		r.sendResponse(r.runner.Status(), nil, conn)
	default:
		r.Debug("handleCommand: unknown command: %s", cmd.Name)
	}
}

func (r *Listener) serve(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var cmd common.Cmd
		if err := json.Unmarshal([]byte(scanner.Text()), &cmd); err != nil {
			r.Debug("serve: invalid JSON: %s", err)
		}
		r.handleCommand(cmd, conn)
	}
	if err := scanner.Err(); err != nil {
		r.Debug("serve: failed to scan: %s", err)
	}
}
