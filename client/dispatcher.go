package client

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
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
	precmd.RemoveOutputFilename()

	cmd := exec.Command(common.DefaultCXX, precmd.GetTokens()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		d.Debug("preprocess failed: %s", string(out[:]))
		return "", errors.Wrap(err, "preprocess failed")
	}
	return string(out[:]), nil
}

func (d *Dispatcher) Run(cmdstr string) error {
	xccmd := common.NewXcodeCmd(cmdstr)
	xccmd.SetArch(runtime.GOARCH)

	outputPath, err := xccmd.GetOutputFilename()
	if err != nil {
		d.Debug("failed to get output path: %s", err)
		return err
	}
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
		fmt.Fprintf(os.Stderr, err.Error())
		return err
	}
	if err := os.WriteFile(outputPath, cmdresp.Object, 0644); err != nil {
		d.Debug("failed to write output file: %s", err)
		return err
	}
	return nil
}
