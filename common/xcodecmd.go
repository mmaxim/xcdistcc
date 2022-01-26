package common

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

type XcodeCmd struct {
	toks []string
}

func NewXcodeCmd(cmd string) *XcodeCmd {
	return &XcodeCmd{
		toks: strings.Split(cmd, " "),
	}
}

func (c *XcodeCmd) Clone() *XcodeCmd {
	ret := new(XcodeCmd)
	ret.toks = make([]string, len(c.toks))
	copy(ret.toks, c.toks)
	return ret
}

func (c *XcodeCmd) GetCommand() string {
	return strings.Join(c.toks, " ")
}

func (c *XcodeCmd) GetTokens() []string {
	return c.toks
}

func (c *XcodeCmd) getSwitchWithArg(name string) (string, error) {
	for index, tok := range c.toks {
		if tok == name && index < len(c.toks)-1 {
			return c.toks[index+1], nil
		}
	}
	return "", errors.New("no switch found")
}

func (c *XcodeCmd) addSwitchWithArg(name, arg string) {
	c.toks = append(c.toks, name)
	c.toks = append(c.toks, arg)
}

func (c *XcodeCmd) removeSwitch(name string, hasArg bool) {
	for index, tok := range c.toks {
		if tok == name {
			stride := 1
			if hasArg {
				stride = 2
			}
			c.toks = append(c.toks[:index], c.toks[index+stride:]...)
			return
		}
	}
}

func (c *XcodeCmd) GetInputFilename() (string, error) {
	arg, err := c.getSwitchWithArg("-c")
	if err != nil {
		return "", errors.Wrap(err, "no input filename switch")
	}
	return filepath.Base(arg), nil
}

func (c *XcodeCmd) GetOutputFilename() (string, error) {
	arg, err := c.getSwitchWithArg("-o")
	if err != nil {
		return "", errors.Wrap(err, "no output filename switch")
	}
	return filepath.Base(arg), nil
}

func (c *XcodeCmd) GetDepFilename() (string, error) {
	arg, err := c.getSwitchWithArg("-MF")
	if err != nil {
		return "", errors.Wrap(err, "no dep filename switch")
	}
	return filepath.Base(arg), nil
}

func (c *XcodeCmd) SetInputFilename(filename string) {
	c.removeSwitch("-c", true)
	c.addSwitchWithArg("-c", filename)
}

func (c *XcodeCmd) SetOutputFilename(filename string) {
	c.removeSwitch("-o", true)
	c.addSwitchWithArg("-o", filename)
}

func (c *XcodeCmd) SetDepFilename(filename string) {
	c.removeSwitch("-MF", true)
	c.addSwitchWithArg("-MF", filename)
}

func (c *XcodeCmd) RemoveOutputFilename() {
	c.removeSwitch("-o", true)
}

func (c *XcodeCmd) RemoveInputFilename() {
	c.removeSwitch("-c", true)
}

func (c *XcodeCmd) RemoveDepFilename() {
	c.removeSwitch("-MF", true)
}

func (c *XcodeCmd) SetPreprocessorOnly() {
	c.toks = append(c.toks, "-E")
}

func (c *XcodeCmd) SetArch(arch string) {
	switch arch {
	case "amd64":
		arch = "x86_64"
	}
	c.removeSwitch("-arch", true)
	c.addSwitchWithArg("-arch", arch)
}

func (c *XcodeCmd) GetArch() string {
	arg, err := c.getSwitchWithArg("-arch")
	if err != nil {
		return runtime.GOARCH
	}
	return arg
}

func (c *XcodeCmd) StripCompiler() {
	c.toks = c.toks[1:]
}
