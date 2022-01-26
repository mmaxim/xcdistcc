package common

import (
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

func (c *XcodeCmd) GetInputFilepath() (string, error) {
	arg, err := c.getSwitchWithArg("-c")
	if err != nil {
		return "", errors.Wrap(err, "no input filepath option")
	}
	return arg, nil
}

func (c *XcodeCmd) GetOutputFilepath() (string, error) {
	arg, err := c.getSwitchWithArg("-o")
	if err != nil {
		return "", errors.Wrap(err, "no output filepath option")
	}
	return arg, nil
}

func (c *XcodeCmd) GetDepFilepath() (string, error) {
	arg, err := c.getSwitchWithArg("-MF")
	if err != nil {
		return "", errors.Wrap(err, "no dep filepath option")
	}
	return arg, nil
}

func (c *XcodeCmd) SetInputFilepath(filepath string) {
	c.removeSwitch("-c", true)
	c.addSwitchWithArg("-c", filepath)
}

func (c *XcodeCmd) SetOutputFilepath(filepath string) {
	c.removeSwitch("-o", true)
	c.addSwitchWithArg("-o", filepath)
}

func (c *XcodeCmd) SetDepFilepath(filepath string) {
	c.removeSwitch("-MF", true)
	c.addSwitchWithArg("-MF", filepath)
}

func (c *XcodeCmd) RemoveOutputFilepath() {
	c.removeSwitch("-o", true)
}

func (c *XcodeCmd) RemoveInputFilepath() {
	c.removeSwitch("-c", true)
}

func (c *XcodeCmd) RemoveDepFilepath() {
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
