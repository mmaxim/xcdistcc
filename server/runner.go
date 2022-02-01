package server

import (
	"sync"

	"mmaxim.org/xcdistcc/common"
)

type jobID uint64

type compileJobRes struct {
	res common.CompileResponse
	err error
}

type compileJob struct {
	id         jobID
	cmd        *common.XcodeCmd
	code       []byte
	includes   []common.IncludeData
	sourceAddr string
	doneCh     chan compileJobRes
}

func newCompileJob(id jobID, cmd common.CompileCmd, sourceAddr string) *compileJob {
	return &compileJob{
		id:         id,
		cmd:        common.NewXcodeCmd(cmd.Command),
		code:       cmd.Code,
		includes:   cmd.Includes,
		sourceAddr: sourceAddr,
		doneCh:     make(chan compileJobRes),
	}
}

func (j *compileJob) toStatusJob() common.StatusJob {
	filename, err := j.cmd.GetInputFilepath()
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

func NewRunner(numWorkers, maxQueueSize int, logger common.Logger) *Runner {
	r := &Runner{
		LabelLogger: common.NewLabelLogger("Runner", logger),
		queue:       newJobQueue[*compileJob](maxQueueSize),
		activeJobs:  make(map[jobID]*compileJob),
	}
	r.Debug("spawning %d workers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go r.compileWorkerLoop(i)
	}
	return r
}

func (r *Runner) compileWorkerLoop(id int) {
	for {
		<-r.queue.wait()
		job, err := r.queue.takeJob()
		if err != nil {
			if err != errNoJobsAvailable {
				r.Debug("failed to take job: %s", err)
			}
			continue
		}
		inputpath, err := job.cmd.GetInputFilepath()
		if err != nil {
			inputpath = "???"
		}
		r.Debug("compiling job: worker: %d input: %s sz: %d queue: %d", id, inputpath,
			len(job.code), len(r.queue.listJobs()))
		builder := NewBuilder(job.code, job.cmd, job.includes, r.GetLogger())
		res, err := builder.Run()
		if err != nil {
			r.Debug("compile failed: %s", err)
		}
		r.Debug("compiling complete: input: %s sz: %d", inputpath, len(res.Object))
		job.doneCh <- compileJobRes{
			res: res,
			err: err,
		}
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
	doneRes := <-job.doneCh
	return doneRes.res, doneRes.err
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
