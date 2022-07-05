package client

import (
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"
	"mmaxim.org/xcdistcc/common"
)

type StatusRemoteSelector struct {
	*common.LabelLogger
	remotes             []Remote
	preprocessorRemotes []Remote
}

func NewStatusRemoteSelector(remotes []Remote, logger common.Logger) *StatusRemoteSelector {
	var prs []Remote
	for _, remote := range remotes {
		if remote.HasPower(PreprocessorPower) {
			prs = append(prs, remote)
		}
	}
	return &StatusRemoteSelector{
		LabelLogger:         common.NewLabelLogger("StatusRemoteSelector", logger),
		remotes:             remotes,
		preprocessorRemotes: prs,
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

func (s *StatusRemoteSelector) bestRemote(remoteScores []remoteScore) (res Remote) {
	bestScore := -1
	for _, rs := range remoteScores {
		if bestScore < 0 || (rs.score >= 0 && rs.score < bestScore) {
			res = rs.remote
			bestScore = rs.score
		}
	}
	return res
}

func (s *StatusRemoteSelector) getBestRemote(remotes []Remote) (res Remote, err error) {
	if len(remotes) == 0 {
		return res, errors.New("no remotes available")
	}
	var scoresMu sync.Mutex
	scores := make([]remoteScore, len(remotes))
	var eg errgroup.Group
	for lindex, lremote := range remotes {
		remote := lremote
		index := lindex
		eg.Go(func() error {
			status, err := s.getRemoteStatus(remote)
			score := -1
			if err != nil {
				s.Debug("GetRemote: failed to get status: %s", err)
			} else {
				score = len(status.QueuedJobs)
			}
			scoresMu.Lock()
			scores[index] = remoteScore{
				score:  score,
				remote: remote,
			}
			scoresMu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return res, err
	}
	return s.bestRemote(scores), nil
}

func (s *StatusRemoteSelector) GetRemote() (res Remote, err error) {
	return s.getBestRemote(s.remotes)
}

func (s *StatusRemoteSelector) GetRemoteWithPreprocessor() (res Remote, err error) {
	return s.getBestRemote(s.preprocessorRemotes)
}
