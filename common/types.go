package common

import "github.com/vmihailenco/msgpack/v5"

type Cmd struct {
	Name string
	Args msgpack.RawMessage
}

type CmdResponse struct {
	Success  bool
	ErrorMsg *string
	Payload  msgpack.RawMessage
}

const MethodCompile = "compile"

type CompileCmd struct {
	Command  string
	Code     []byte
	Includes []IncludeData
}

type CompileResponse struct {
	Output string
	Object []byte
	Dep    []byte
}

const MethodPreprocess = "preprocess"

type PreprocessCmd struct {
	Dir     string
	Command string
}

type PreprocessResponse struct {
	Output string
	Code   []byte
	Dep    []byte
}

const MethodStatus = "status"

type StatusCmd struct{}

type StatusJob struct {
	SourceAddress string
	Filename      string
	Command       string
	Mode          string
}

type StatusResponse struct {
	ActiveJobs []StatusJob
	QueuedJobs []StatusJob
}

type IncludeData struct {
	Path string
	Data string
}
