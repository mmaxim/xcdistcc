package client

import (
	"os/exec"
	"runtime"

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

func (d *Dispatcher) preprocess(basecmd *common.XcodeCmd) (string, error) {
	precmd := new(common.XcodeCmd)
	*precmd = *basecmd
	precmd.SetPreprocessorOnly()
	precmd.RemoveOutputFilename()

	d.Debug(precmd.GetCommand())
	cmd := exec.Command(common.DefaultCXX, precmd.GetTokens()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "preprocess failed")
	}
	return string(out[:]), nil
}

func (d *Dispatcher) Run(cmd string) error {
	xccmd := common.NewXcodeCmd(cmd)
	xccmd.SetArch(runtime.GOARCH)
	xccmd.StripCompiler()

	preprocessed, err := d.preprocess(xccmd)
	if err != nil {
		d.Debug("failed to preprocess: %s", err)
		return err
	}

	d.Debug(preprocessed)
	return nil
}
