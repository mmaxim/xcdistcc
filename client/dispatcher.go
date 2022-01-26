package client

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/common"
)

type Dispatcher struct {
	*common.LabelLogger
	hosts []string
}

func NewDispatcher(hosts []string) *Dispatcher {
	return &Dispatcher{
		LabelLogger: common.NewLabelLogger("Dispatcher"),
		hosts:       hosts,
	}
}

func (d *Dispatcher) getConn() (net.Conn, error) {
	host := d.hosts[rand.Intn(len(d.hosts))]
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, common.DefaultListenPort)
	}
	return net.Dial("tcp", host)
}

func (d *Dispatcher) preprocess(basecmd *common.XcodeCmd) (string, error) {
	precmd := basecmd.Clone()
	precmd.StripCompiler()
	precmd.SetPreprocessorOnly()
	precmd.RemoveOutputFilepath()

	cmd := exec.Command(common.DefaultCXX, precmd.GetTokens()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		d.Debug("preprocess failed: %s", string(out[:]))
		return "", errors.Wrap(err, "preprocess failed")
	}
	return string(out[:]), nil
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
	d.Debug("output path: %s", outputPath)
	preprocessed, err := d.preprocess(xccmd)
	if err != nil {
		d.Debug("failed to preprocess: %s", err)
		return err
	}

	conn, err := d.getConn()
	if err != nil {
		d.Debug("failed to get runner connection: %s", err)
		return err
	}
	var cmdresp common.CompileResponse
	if cmdresp, err = common.DoRPC[common.CompileCmd, common.CompileResponse](conn, common.MethodCompile,
		common.CompileCmd{
			Command: xccmd.GetCommand(),
			Code:    preprocessed,
		}); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return err
	}

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
	return nil
}
