package ui

import (
	"fmt"
	"path/filepath"

	"mmaxim.org/xcdistcc/client"
	"mmaxim.org/xcdistcc/common"
)

type Refresher struct {
	remotes []client.Remote
}

func NewRefresher(remotes []client.Remote) *Refresher {
	return &Refresher{
		remotes: remotes,
	}
}

func (r *Refresher) getStatus(remote client.Remote) (res []string, err error) {
	conn, err := client.DialRemote(remote)
	if err != nil {
		return nil, err
	}
	status, err := common.DoRPC[common.StatusCmd, common.StatusResponse](conn.Conn, common.MethodStatus,
		common.StatusCmd{}, conn.Secret)
	if err != nil {
		return nil, err
	}
	for _, status := range status.WorkerStatus {
		mode := "Idle"
		if status.Job != nil {
			mode = status.Job.Mode
			switch mode {
			case "Compile":
				mode += fmt.Sprintf(": %s", filepath.Base(status.Job.Filename))
			}
		}
		res = append(res, fmt.Sprintf("worker: %d addr: %s mode: %s", status.ID, remote.Address, mode))
	}
	return res, nil
}

func (r *Refresher) GetStatuses() (res []string, err error) {
	for _, remote := range r.remotes {
		jobStrs, err := r.getStatus(remote)
		if err != nil {
			continue
		}
		res = append(res, jobStrs...)
	}
	return res, nil
}
