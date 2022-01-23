package server

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/common"
)

type Builder struct {
	*common.LabelLogger

	code string
	cmd  *common.XcodeCmd
}

func NewBuilder(code string, cmd *common.XcodeCmd) *Builder {
	return &Builder{
		LabelLogger: common.NewLabelLogger("Builder"),
		code:        code,
		cmd:         cmd,
	}
}

func (b *Builder) Run() (res common.CompileResponse, err error) {
	// make temp directory for our job
	dir := os.TempDir()
	if err := os.MkdirAll(dir, 0644); err != nil {
		return res, errors.Wrap(err, "failed to make temp dir")
	}
	defer os.RemoveAll(dir)

	// write out temp input file with same name
	inputFilename, err := b.cmd.GetInputFilename()
	if err != nil {
		return res, err
	}
	if err := os.WriteFile(filepath.Join(dir, inputFilename), []byte(b.code), 0644); err != nil {
		return res, errors.Wrap(err, "failed to write input file")
	}

	ccmd := new(common.XcodeCmd)
	*ccmd = *b.cmd
	ccmd.StripCompiler()
	b.Debug("%v", ccmd.GetTokens())
	ecmd := exec.Command(common.DefaultCXX, ccmd.GetTokens()...)
	ecmd.Dir = dir
	out, err := ecmd.CombinedOutput()
	if err != nil {
		b.Debug("failed to run command: out: %s err: %s", out, err)
		return res, errors.Wrap(err, "failed to run compile command")
	}

	// read output file
	outputFilename, err := b.cmd.GetOutputFilename()
	if err != nil {
		return res, err
	}
	object, err := os.ReadFile(filepath.Join(dir, outputFilename))
	if err != nil {
		return res, errors.Wrap(err, "failed to read object file")
	}

	res.Output = string(out)
	res.Object = object
	return res, nil
}
