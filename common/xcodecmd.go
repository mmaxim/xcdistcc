package common

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
)

type XcodeCmd struct {
	toks []string
}

func NewXcodeCmd(cmd string) *XcodeCmd {
	return &XcodeCmd{
		toks: strings.Split(cmd, " "),
	}
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
	return "", errors.New("no input filename given")
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
			c.toks = append(c.toks[:index-1], c.toks[index+stride:]...)
			return
		}
	}
}

func (c *XcodeCmd) GetInputFilename() (string, error) {
	arg, err := c.getSwitchWithArg("-c")
	if err != nil {
		return "", err
	}
	return filepath.Base(arg), nil
}

func (c *XcodeCmd) GetOutputFilename() (string, error) {
	arg, err := c.getSwitchWithArg("-o")
	if err != nil {
		return "", err
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

func (c *XcodeCmd) RemoveOutputFilename() {
	c.removeSwitch("-o", true)
}

func (c *XcodeCmd) SetPreprocessorOnly() {
	c.toks = append(c.toks, "-E")
}

func (c *XcodeCmd) SetArch(arch string) {
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
