package client

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/common"
)

type Remote struct {
	Address      string
	PublicKeyStr string
}

type RemoteSelector interface {
	GetRemote() (Remote, error)
}

type Preprocessor interface {
	Preprocess(cmd *common.XcodeCmd) ([]byte, *common.XcodeCmd, []common.IncludeData, error)
}

type Dispatcher struct {
	*common.LabelLogger
	remoteSelector RemoteSelector
	preprocessor   Preprocessor
}

func NewDispatcher(remoteSelector RemoteSelector, preprocessor Preprocessor, logger common.Logger) *Dispatcher {
	return &Dispatcher{
		LabelLogger:    common.NewLabelLogger("Dispatcher", logger),
		remoteSelector: remoteSelector,
		preprocessor:   preprocessor,
	}
}

func (d *Dispatcher) getConn() (*RemoteConn, error) {
	remote, err := d.remoteSelector.GetRemote()
	if err != nil {
		return nil, err
	}
	return DialRemote(remote)
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
	preprocessed, precmd, includeData, err := d.preprocessor.Preprocess(xccmd)
	if err != nil {
		d.Debug("failed to preprocess: %s", err)
		return err
	}
	xccmd = precmd
	d.Debug("preprocessing done: %s sz: %d sdur: %v tdur: %v", outputPath, len(preprocessed),
		time.Since(stageTime), time.Since(startTime))

	conn, err := d.getConn()
	if err != nil {
		d.Debug("failed to get runner connection: %s", err)
		return err
	}
	stageTime = time.Now()
	var cmdresp common.CompileResponse
	if cmdresp, err = common.DoRPC[common.CompileCmd, common.CompileResponse](conn.Conn, common.MethodCompile,
		common.CompileCmd{
			Command:  xccmd.GetCommand(),
			Code:     preprocessed,
			Includes: includeData,
		}, conn.Secret); err != nil {
		d.Debug("failed to compile")
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
	if err := d.writeFile(outputPath, cmdresp.Object); err != nil {
		d.Debug("failed to write output file: %s", err)
		return err
	}
	d.Debug("write done: %s sdur: %v tdur: %v", outputPath, time.Since(stageTime), time.Since(startTime))
	return nil
}
