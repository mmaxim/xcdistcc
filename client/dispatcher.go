package client

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/common"
)

type HostSelector interface {
	GetHost() (string, error)
}

type Dispatcher struct {
	*common.LabelLogger
	hostSelector HostSelector
}

func NewDispatcher(hostSelector HostSelector, logger common.Logger) *Dispatcher {
	return &Dispatcher{
		LabelLogger:  common.NewLabelLogger("Dispatcher", logger),
		hostSelector: hostSelector,
	}
}

func (d *Dispatcher) getConn() (net.Conn, error) {
	host, err := d.hostSelector.GetHost()
	if err != nil {
		return nil, err
	}
	return net.Dial("tcp", host)
}

func (d *Dispatcher) preprocess(basecmd *common.XcodeCmd) ([]byte, error) {
	precmd := basecmd.Clone()
	precmd.StripCompiler()
	precmd.SetPreprocessorOnly()
	precmd.RemoveOutputFilepath()

	cmd := exec.Command(common.DefaultCXX, precmd.GetTokens()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		d.Debug("preprocess failed: %s", string(out[:]))
		return nil, errors.Wrap(err, "preprocess failed")
	}
	var gzipBuf bytes.Buffer
	compressor := gzip.NewWriter(&gzipBuf)
	if _, err := io.Copy(compressor, bytes.NewBuffer(out)); err != nil {
		return nil, errors.Wrap(err, "failed to compress")
	}
	if err := compressor.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close compressor")
	}
	return gzipBuf.Bytes(), nil
}

func (d *Dispatcher) writeFile(fullpath string, dat []byte) error {
	if err := os.MkdirAll(filepath.Dir(fullpath), 0644); err != nil {
		return errors.Wrap(err, "failed to make directory")
	}
	if err := os.WriteFile(fullpath, dat, 0644); err != nil {
		return errors.Wrap(err, "failed to write file")
	}
	return nil
}

func (d *Dispatcher) Run(cmdstr string) error {
	xccmd := common.NewXcodeCmd(cmdstr)
	xccmd.SetArch(runtime.GOARCH)

	outputPath, err := xccmd.GetOutputFilepath()
	if err != nil {
		d.Debug("failed to get output path: %s", err)
		return err
	}
	startTime := time.Now()
	stageTime := time.Now()
	preprocessed, err := d.preprocess(xccmd)
	if err != nil {
		d.Debug("failed to preprocess: %s", err)
		return err
	}
	xccmd.RemoveDepFilepath()
	d.Debug("preprocessing done: %s sz: %d sdur: %v tdur: %v", outputPath, len(preprocessed),
		time.Since(stageTime), time.Since(startTime))

	conn, err := d.getConn()
	if err != nil {
		d.Debug("failed to get runner connection: %s", err)
		return err
	}
	stageTime = time.Now()
	var cmdresp common.CompileResponse
	if cmdresp, err = common.DoRPC[common.CompileCmd, common.CompileResponse](conn, common.MethodCompile,
		common.CompileCmd{
			Command: xccmd.GetCommand(),
			Code:    base64.StdEncoding.EncodeToString(preprocessed),
		}); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return err
	}
	d.Debug("compile done: %s sdur: %v tdur: %v", outputPath, time.Since(stageTime), time.Since(startTime))

	stageTime = time.Now()
	// write dep file if one was specified
	depPath, err := xccmd.GetDepFilepath()
	if err == nil {
		if err := d.writeFile(depPath, cmdresp.Dep); err != nil {
			d.Debug("failed to write dep file: %s", err)
			return err
		}
	}
	// ensure output path exists
	if err := d.writeFile(outputPath, cmdresp.Object); err != nil {
		d.Debug("failed to write output file: %s", err)
		return err
	}
	d.Debug("write done: %s sdur: %v tdur: %v", outputPath, time.Since(stageTime), time.Since(startTime))
	return nil
}
