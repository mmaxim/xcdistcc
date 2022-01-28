package server

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

func NewBuilder(code string, cmd *common.XcodeCmd, logger common.Logger) *Builder {
	return &Builder{
		LabelLogger: common.NewLabelLogger("Builder", logger),
		code:        code,
		cmd:         cmd,
	}
}

func (b *Builder) decodeCode() (string, error) {
	out, err := base64.StdEncoding.DecodeString(b.code)
	if err != nil {
		return "", errors.Wrap(err, "failed to base64 decode")
	}
	gzipReader, err := gzip.NewReader(bytes.NewBuffer(out))
	if err != nil {
		return "", errors.Wrap(err, "failed to make gzip reader")
	}
	defer gzipReader.Close()
	code, err := io.ReadAll(gzipReader)
	if err != nil {
		return "", errors.Wrap(err, "failed to decompress")
	}
	return string(code[:]), nil
}

func (b *Builder) Run() (res common.CompileResponse, err error) {
	code, err := b.decodeCode()
	if err != nil {
		return res, err
	}

	owndir, err := common.RandString("xc", 9)
	if err != nil {
		return res, errors.Wrap(err, "failed to generate build dir name")
	}
	// make temp directory for our job
	dir := filepath.Join(os.TempDir(), owndir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return res, errors.Wrap(err, "failed to make temp dir")
	}
	defer os.RemoveAll(dir)
	ccmd := b.cmd.Clone()

	// write out temp input file with same name
	origInputPath, err := b.cmd.GetInputFilepath()
	if err != nil {
		return res, err
	}
	inputFilepath := filepath.Join(dir, filepath.Base(origInputPath))
	if err := os.WriteFile(inputFilepath, []byte(code), 0644); err != nil {
		return res, errors.Wrap(err, "failed to write input file")
	}
	ccmd.RemoveInputFilepath()
	ccmd.SetInputFilepath(inputFilepath)

	origOutputPath, err := b.cmd.GetOutputFilepath()
	if err != nil {
		return res, err
	}
	outputFilepath := filepath.Join(dir, filepath.Base(origOutputPath))
	ccmd.RemoveOutputFilepath()
	ccmd.SetOutputFilepath(outputFilepath)

	var depFilepath string
	origDepPath, err := b.cmd.GetDepFilepath()
	if err != nil {
		b.Debug("no depfile, skipping creating it")
	} else {
		depFilepath = filepath.Join(dir, filepath.Base(origDepPath))
		ccmd.RemoveDepFilepath()
		ccmd.SetDepFilepath(depFilepath)
	}

	ccmd.StripCompiler()
	//b.Debug("compile command: %s", ccmd.GetCommand())
	ecmd := exec.Command(common.DefaultCXX, ccmd.GetTokens()...)
	out, err := ecmd.CombinedOutput()
	if err != nil {
		b.Debug("failed to run command: out: %s err: %s", out, err)
		return res, newCompileError(err.Error(), string(out[:]))
	}

	// read output file
	object, err := os.ReadFile(outputFilepath)
	if err != nil {
		return res, errors.Wrap(err, "failed to read object file")
	}
	// read dep file if requested
	if len(depFilepath) != 0 {
		dep, err := os.ReadFile(depFilepath)
		if err != nil {
			return res, errors.Wrap(err, "failed to read dep file")
		}
		// these files get our modified path in them, so replace that path with the original so they work on
		// the host machine
		res.Dep = []byte(strings.ReplaceAll(string(dep[:]), inputFilepath, origInputPath))
	}
	res.Output = string(out)
	res.Object = object
	return res, nil
}
