package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/client"
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

	preprocessor *client.ClangPreprocessor
}

func NewBuilder(logger common.Logger) *Builder {
	return &Builder{
		LabelLogger:  common.NewLabelLogger("Builder", logger),
		preprocessor: client.NewClangPreprocessor(logger),
	}
}

func (b *Builder) Compile(code []byte, cmd *common.XcodeCmd, includes []common.IncludeData) (res common.CompileResponse, err error) {
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
	ccmd := cmd.Clone()

	// write out temp input file with same name
	origInputPath, err := cmd.GetInputFilepath()
	if err != nil {
		return res, err
	}
	inputFilepath := filepath.Join(dir, filepath.Base(origInputPath))
	if err := os.WriteFile(inputFilepath, code, 0644); err != nil {
		return res, errors.Wrap(err, "failed to write input file")
	}
	ccmd.RemoveInputFilepath()
	ccmd.SetInputFilepath(inputFilepath)

	origOutputPath, err := cmd.GetOutputFilepath()
	if err != nil {
		return res, err
	}
	outputFilepath := filepath.Join(dir, filepath.Base(origOutputPath))
	ccmd.RemoveOutputFilepath()
	ccmd.SetOutputFilepath(outputFilepath)

	var depFilepath string
	origDepPath, err := cmd.GetDepFilepath()
	if err == nil {
		depFilepath = filepath.Join(dir, filepath.Base(origDepPath))
		ccmd.RemoveDepFilepath()
		ccmd.SetDepFilepath(depFilepath)
	}

	// if we have include data, create the localized version of it in the temp dir, and change the
	// compile commands be rooted in it
	if len(includes) != 0 {
		for _, include := range includes {
			dest := dir + include.Path
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				b.Debug("failed to make include dirs: dest: %s err: %s", dest, err)
				continue
			}
			if err := os.WriteFile(dest, []byte(include.Data), 0644); err != nil {
				b.Debug("failed to write include: dest: %s err: %s", dest, err)
				continue
			}
		}
		ccmd.LocalizeIncludeDirs(dir)
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

func (b *Builder) Preprocess(dir string, cmd *common.XcodeCmd) (res common.PreprocessResponse, err error) {
	b.preprocessor.SetDirectory(dir)
	out, _, _, err := b.preprocessor.Preprocess(cmd)
	if err != nil {
		return res, err
	}
	res.Code = out
	depFilepath, err := cmd.GetDepFilepath()
	if err == nil {
		var absDepFilepath string
		if filepath.IsAbs(depFilepath) {
			absDepFilepath = depFilepath
		} else {
			absDepFilepath = dir + string(filepath.Separator) + depFilepath
		}
		dep, err := os.ReadFile(absDepFilepath)
		if err != nil {
			return res, errors.Wrap(err, "failed to read dep file")
		}
		res.Dep = dep
	}
	return res, nil
}
