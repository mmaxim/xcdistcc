package server

import (
	"sync"

	"mmaxim.org/xcdistcc/common"
)

type jobID uint64

type compileJob struct {
	id         jobID
	cmd        *common.XcodeCmd
	code       string
	sourceAddr string
	doneCh     chan common.CompileResponse
}

func newCompileJob(id jobID, cmd common.CompileCmd, sourceAddr string) *compileJob {
	return &compileJob{
		id:         id,
		cmd:        common.NewXcodeCmd(cmd.Command),
		code:       cmd.Code,
		sourceAddr: sourceAddr,
		doneCh:     make(chan common.CompileResponse),
	}
}

func (j *compileJob) toStatusJob() common.StatusJob {
	filename, err := j.cmd.GetInputFilename()
	if err != nil {
		filename = "???"
	}
	return common.StatusJob{
		SourceAddress: j.sourceAddr,
		Filename:      filename,
		Command:       j.cmd.GetCommand(),
	}
}

type Runner struct {
	*common.LabelLogger
	queue *jobQueue[*compileJob]

	activeJobsMu sync.Mutex
	nextJobID    jobID
	activeJobs   map[jobID]*compileJob
}

func NewRunner(numWorkers, maxQueueSize int) *Runner {
	r := &Runner{
		LabelLogger: common.NewLabelLogger("Runner"),
		queue:       newJobQueue[*compileJob](maxQueueSize),
		activeJobs:  make(map[jobID]*compileJob),
	}
	for i := 0; i < numWorkers; i++ {
		go r.compileWorkerLoop()
	}
	return r
}

func (r *Runner) compileWorkerLoop() {
	for {
		<-r.queue.wait()
		job, err := r.queue.takeJob()
		if err != nil {
			if err != errNoJobsAvailable {
				r.Debug("failed to take job: %s", err)
			}
			continue
		}
		r.Debug("compiling job: %s", job.cmd.GetCommand())
		r.finishCompileJob(job)
	}
}

func (r *Runner) queueCompileJob(cmd common.CompileCmd, sourceAddr string) (*compileJob, error) {
	r.activeJobsMu.Lock()
	defer r.activeJobsMu.Unlock()
	r.nextJobID += 1
	job := newCompileJob(r.nextJobID, cmd, sourceAddr)
	if err := r.queue.push(job); err != nil {
		return job, err
	}
	return job, nil
}

func (r *Runner) finishCompileJob(job *compileJob) {
	r.activeJobsMu.Lock()
	defer r.activeJobsMu.Unlock()
	delete(r.activeJobs, job.id)
}

func (r *Runner) Compile(cmd common.CompileCmd, sourceAddr string) (res common.CompileResponse, err error) {
	job, err := r.queueCompileJob(cmd, sourceAddr)
	if err != nil {
		return res, err
	}
	return <-job.doneCh, nil
}

func (r *Runner) Status() (res common.StatusResponse) {
	r.activeJobsMu.Lock()
	defer r.activeJobsMu.Unlock()
	for _, job := range r.activeJobs {
		res.ActiveJobs = append(res.ActiveJobs, job.toStatusJob())
	}
	jobs := r.queue.listJobs()
	for _, job := range jobs {
		res.QueuedJobs = append(res.QueuedJobs, job.toStatusJob())
	}
	return res
}
