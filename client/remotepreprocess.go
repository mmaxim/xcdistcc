package client

import (
	"os"

	"mmaxim.org/xcdistcc/common"
)

type RemotePreprocessor struct {
	*common.LabelLogger
	remoteSelector RemoteSelector
	backup         Preprocessor
}

func NewRemotePreprocessor(remoteSelector RemoteSelector, backup Preprocessor, logger common.Logger) *RemotePreprocessor {
	return &RemotePreprocessor{
		LabelLogger:    common.NewLabelLogger("RemotePreprocessor", logger),
		remoteSelector: remoteSelector,
		backup:         backup,
	}
}

func (p *RemotePreprocessor) getConn() (*RemoteConn, error) {
	remote, err := p.remoteSelector.GetRemoteWithPreprocessor()
	if err != nil {
		return nil, err
	}
	return DialRemote(remote)
}

func (p *RemotePreprocessor) Preprocess(cmd *common.XcodeCmd) (res []byte, retcmd *common.XcodeCmd, includes []common.IncludeData, err error) {
	defer func() {
		if err != nil {
			p.Debug("failed to get remote conn, using backup: %s", err)
			res, retcmd, includes, err = p.backup.Preprocess(cmd)
		}
	}()

	conn, err := p.getConn()
	if err != nil {
		return res, retcmd, includes, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return res, retcmd, includes, err
	}

	var cmdresp common.PreprocessResponse
	if cmdresp, err = common.DoRPC[common.PreprocessCmd, common.PreprocessResponse](conn.Conn,
		common.MethodPreprocess,
		common.PreprocessCmd{
			Dir:     wd,
			Command: cmd.GetCommand(),
		}, conn.Secret); err != nil {
		return res, retcmd, includes, err
	}
	depPath, err := cmd.GetDepFilepath()
	if err == nil {
		if err := common.WriteFileCreatePath(depPath, cmdresp.Dep); err != nil {
			p.Debug("failed to write dep file: %s", err)
			return res, retcmd, includes, err
		}
	}
	retcmd = cmd.Clone()
	retcmd.RemoveDepFilepath()

	return cmdresp.Code, retcmd, nil, nil
}
