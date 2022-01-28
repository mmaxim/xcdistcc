package client

import (
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
	"mmaxim.org/xcdistcc/common"
)

type StatusHostSelector struct {
	*common.LabelLogger
	hosts []string
}

func NewStatusHostSelector(hosts []string, logger common.Logger) *StatusHostSelector {
	return &StatusHostSelector{
		LabelLogger: common.NewLabelLogger("StatusHostSelector", logger),
		hosts:       hosts,
	}
}

func (s *StatusHostSelector) getHostStatus(host string) (res common.StatusResponse, err error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return res, err
	}
	return common.DoRPC[common.StatusCmd, common.StatusResponse](conn, common.MethodStatus,
		common.StatusCmd{})
}

func (s *StatusHostSelector) bestHost(queueSizes map[string]int) (res string) {
	bestQueueSize := -1
	for host, queueSize := range queueSizes {
		if queueSize > bestQueueSize {
			res = host
			bestQueueSize = queueSize
		}
	}
	return res
}

func (s *StatusHostSelector) GetHost() (string, error) {
	var queueSizesMu sync.Mutex
	queueSizes := make(map[string]int)
	var eg errgroup.Group
	for _, lhost := range s.hosts {
		host := lhost
		eg.Go(func() error {
			status, err := s.getHostStatus(host)
			if err != nil {
				s.Debug("GetHost: failed to get status: %s", err)
				return err
			}
			queueSizesMu.Lock()
			queueSizes[host] = len(status.QueuedJobs)
			queueSizesMu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return "", err
	}
	return s.bestHost(queueSizes), nil
}
