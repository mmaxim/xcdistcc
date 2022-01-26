package server

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/common"
)

type compileError struct {
	msg    string
	output string
}

func newCompileError(msg, output string) compileError {
	return compileError{
		msg:    msg,
		output: output,
	}
}

func (e compileError) Msg() string {
	return e.msg
}

func (e compileError) Error() string {
	return e.output
}

// =============================================================================

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
	ccmd := b.cmd.Clone()

	// write out temp input file with same name
	inputPath, err := b.cmd.GetInputFilepath()
	if err != nil {
		return res, err
	}
	inputFilename := filepath.Base(inputPath)
	if err := os.WriteFile(filepath.Join(dir, inputFilename), []byte(b.code), 0644); err != nil {
		return res, errors.Wrap(err, "failed to write input file")
	}
	ccmd.RemoveInputFilepath()
	ccmd.SetInputFilepath(inputFilename)

	outputPath, err := b.cmd.GetOutputFilepath()
	if err != nil {
		return res, err
	}
	outputFilename := filepath.Base(outputPath)
	ccmd.RemoveOutputFilepath()
	ccmd.SetOutputFilepath(outputFilename)

	var depFilename string
	depPath, err := b.cmd.GetDepFilepath()
	if err != nil {
		b.Debug("no depfile, skipping creating it")
	} else {
		depFilename = filepath.Base(depPath)
		ccmd.RemoveDepFilepath()
		ccmd.SetDepFilepath(depFilename)
	}

	ccmd.StripCompiler()
	b.Debug("compile command: %s", ccmd.GetCommand())
	ecmd := exec.Command(common.DefaultCXX, ccmd.GetTokens()...)
	ecmd.Dir = dir
	out, err := ecmd.CombinedOutput()
	if err != nil {
		b.Debug("failed to run command: out: %s err: %s", out, err)
		return res, newCompileError(err.Error(), string(out[:]))
	}

	// read output file
	object, err := os.ReadFile(filepath.Join(dir, outputFilename))
	if err != nil {
		return res, errors.Wrap(err, "failed to read object file")
	}
	// read dep file if requested
	if len(depFilename) != 0 {
		dep, err := os.ReadFile(filepath.Join(dir, depFilename))
		if err != nil {
			return res, errors.Wrap(err, "failed to read dep file")
		}
		res.Dep = dep
	}
	res.Output = string(out)
	res.Object = object
	return res, nil
}
