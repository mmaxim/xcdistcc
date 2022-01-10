package common

import "encoding/json"

type Cmd struct {
	Name string
	Args json.RawMessage
}

type CmdResponse struct {
	Success  bool
	ErrorMsg *string
	Payload  json.RawMessage
}

const MethodCompile = "compile"

type CompileCmd struct {
	Command string
	Code    string
}

type CompileResponse struct {
	Output string
	Object []byte // blank if there is a compile error
}

const MethodStatus = "status"

type StatusCmd struct{}

type StatusJob struct {
	SourceAddress string
	Filename      string
	Command       string
}

type StatusResponse struct {
	ActiveJobs []StatusJob
	QueuedJobs []StatusJob
}
