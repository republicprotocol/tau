package vm

import (
	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/process"
)

type Exec struct {
	proc process.Process
}

func NewExec(proc process.Process) Exec {
	return Exec{
		proc,
	}
}

func (message Exec) IsMessage() {
}

type Result struct {
	Value process.Value
}

func NewResult(value process.Value) Result {
	return Result{value}
}

func (message Result) IsMessage() {
}

type RemoteProcedureCall struct {
	Message buffer.Message
}

func NewRemoteProcedureCall(message buffer.Message) RemoteProcedureCall {
	return RemoteProcedureCall{
		message,
	}
}

func (message RemoteProcedureCall) IsMessage() {
}
