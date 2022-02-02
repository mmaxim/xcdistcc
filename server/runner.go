package server

import (
	"sync"

	"mmaxim.org/xcdistcc/common"
)

type jobID uint64

type runnerJob interface {
	toStatusJob() common.StatusJob
	ID() jobID
}

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

func (j *compileJob) ID() jobID {
	return j.id
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
		Mode:          "Compile",
	}
}

type preprocessJobRes struct {
	res common.PreprocessResponse
	err error
}

type preprocessJob struct {
	id         jobID
	dir        string
	cmd        *common.XcodeCmd
	sourceAddr string
	doneCh     chan preprocessJobRes
}

func newPreprocessJob(id jobID, cmd common.PreprocessCmd, sourceAddr string) *preprocessJob {
	return &preprocessJob{
		id:         id,
		dir:        cmd.Dir,
		cmd:        common.NewXcodeCmd(cmd.Command),
		sourceAddr: sourceAddr,
		doneCh:     make(chan preprocessJobRes),
	}
}

func (j *preprocessJob) ID() jobID {
	return j.id
}

func (j *preprocessJob) toStatusJob() common.StatusJob {
	filename, err := j.cmd.GetInputFilepath()
	if err != nil {
		filename = "???"
	}
	return common.StatusJob{
		SourceAddress: j.sourceAddr,
		Filename:      filename,
		Command:       j.cmd.GetCommand(),
		Mode:          "Preprocess",
	}
}

type Runner struct {
	*common.LabelLogger
	queue   *jobQueue[runnerJob]
	builder *Builder

	activeJobsMu sync.Mutex
	nextJobID    jobID
	activeJobs   map[jobID]runnerJob
}

func NewRunner(numWorkers, maxQueueSize int, logger common.Logger) *Runner {
	r := &Runner{
		LabelLogger: common.NewLabelLogger("Runner", logger),
		queue:       newJobQueue[runnerJob](maxQueueSize),
		builder:     NewBuilder(logger),
		activeJobs:  make(map[jobID]runnerJob),
	}
	r.Debug("spawning %d workers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go r.workerLoop(i)
	}
	return r
}

func (r *Runner) runCompileJob(job *compileJob) {
	inputpath, err := job.cmd.GetInputFilepath()
	if err != nil {
		inputpath = "???"
	}
	r.Debug("compiling job: input: %s sz: %d queue: %d", inputpath,
		len(job.code), len(r.queue.listJobs()))
	res, err := r.builder.Compile(job.code, job.cmd, job.includes)
	if err != nil {
		r.Debug("compile failed: %s", err)
	}
	r.Debug("compiling complete: input: %s sz: %d", inputpath, len(res.Object))
	job.doneCh <- compileJobRes{
		res: res,
		err: err,
	}
}

func (r *Runner) runPreprocessJob(job *preprocessJob) {
	inputpath, err := job.cmd.GetInputFilepath()
	if err != nil {
		inputpath = "???"
	}
	r.Debug("preprocessing job: input: %s dir: %s queue: %d", inputpath, job.dir, len(r.queue.listJobs()))
	res, err := r.builder.Preprocess(job.dir, job.cmd)
	if err != nil {
		r.Debug("preprocess failed: %s", err)
	}
	r.Debug("preprocessing complete: input: %s sz: %d", inputpath, len(res.Code))
	job.doneCh <- preprocessJobRes{
		res: res,
		err: err,
	}
}

func (r *Runner) workerLoop(id int) {
	for {
		<-r.queue.wait()
		job, err := r.queue.takeJob()
		if err != nil {
			if err != errNoJobsAvailable {
				r.Debug("failed to take job: %s", err)
			}
			continue
		}
		switch sjob := job.(type) {
		case *compileJob:
			r.runCompileJob(sjob)
		case *preprocessJob:
			r.runPreprocessJob(sjob)
		default:
			r.Debug("unknown job type")
		}

		r.finishCompileJob(job)
	}
}

func (r *Runner) getJobID() jobID {
	r.activeJobsMu.Lock()
	defer r.activeJobsMu.Unlock()
	r.nextJobID += 1
	return r.nextJobID
}

func (r *Runner) finishCompileJob(job runnerJob) {
	r.activeJobsMu.Lock()
	defer r.activeJobsMu.Unlock()
	delete(r.activeJobs, job.ID())
}

func (r *Runner) Compile(cmd common.CompileCmd, sourceAddr string) (res common.CompileResponse, err error) {
	job := newCompileJob(r.getJobID(), cmd, sourceAddr)
	if err := r.queue.push(job); err != nil {
		return res, err
	}
	doneRes := <-job.doneCh
	return doneRes.res, doneRes.err
}

func (r *Runner) Preprocess(cmd common.PreprocessCmd, sourceAddr string) (res common.PreprocessResponse, err error) {
	job := newPreprocessJob(r.getJobID(), cmd, sourceAddr)
	if err := r.queue.push(job); err != nil {
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
