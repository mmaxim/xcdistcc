package client

import (
	"sync"

	"golang.org/x/sync/errgroup"
	"mmaxim.org/xcdistcc/common"
)

type StatusRemoteSelector struct {
	*common.LabelLogger
	remotes []Remote
}

func NewStatusHostSelector(remotes []Remote, logger common.Logger) *StatusRemoteSelector {
	return &StatusRemoteSelector{
		LabelLogger: common.NewLabelLogger("StatusRemoteSelector", logger),
		remotes:     remotes,
	}
}

func (s *StatusRemoteSelector) getRemoteStatus(remote Remote) (res common.StatusResponse, err error) {
	conn, err := DialRemote(remote)
	if err != nil {
		return res, err
	}
	return common.DoRPC[common.StatusCmd, common.StatusResponse](conn.Conn, common.MethodStatus,
		common.StatusCmd{}, conn.Secret)
}

type remoteScore struct {
	score  int
	remote Remote
}

func (s *StatusRemoteSelector) bestRemote(remoteScores map[string]remoteScore) (res Remote) {
	bestScore := -1
	for _, rs := range remoteScores {
		if rs.score > bestScore {
			res = rs.remote
			bestScore = rs.score
		}
	}
	return res
}

func (s *StatusRemoteSelector) GetRemote() (res Remote, err error) {
	var queueSizesMu sync.Mutex
	queueSizes := make(map[string]remoteScore)
	var eg errgroup.Group
	for _, lremote := range s.remotes {
		remote := lremote
		eg.Go(func() error {
			status, err := s.getRemoteStatus(remote)
			if err != nil {
				s.Debug("GetRemote: failed to get status: %s", err)
				return err
			}
			queueSizesMu.Lock()
			queueSizes[remote.Address] = remoteScore{
				score:  len(status.QueuedJobs),
				remote: remote,
			}
			queueSizesMu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return res, err
	}
	return s.bestRemote(queueSizes), nil
}
