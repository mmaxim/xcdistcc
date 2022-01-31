package client

import (
	"os/exec"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/common"
)

type ClangPreprocessor struct {
	*common.LabelLogger
}

func NewClangPreprocessor(logger common.Logger) *ClangPreprocessor {
	return &ClangPreprocessor{
		LabelLogger: common.NewLabelLogger("ClangPreprocessor", logger),
	}
}

func (c *ClangPreprocessor) Preprocess(basecmd *common.XcodeCmd) ([]byte, *common.XcodeCmd, []common.IncludeData, error) {
	precmd := basecmd.Clone()
	retcmd := basecmd.Clone()
	precmd.StripCompiler()
	precmd.SetPreprocessorOnly()
	precmd.RemoveOutputFilepath()

	cmd := exec.Command(common.DefaultCXX, precmd.GetTokens()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		c.Debug("preprocess failed: %s", string(out[:]))
		return nil, nil, nil, errors.Wrap(err, "preprocess failed")
	}
	retcmd.RemoveDepFilepath()
	return out, retcmd, nil, nil

}
