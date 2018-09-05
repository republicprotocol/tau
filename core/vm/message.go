package vm

import (
	"github.com/republicprotocol/smpc-go/core/process"
)

type Exec struct {
	proc process.Process
}

func NewExecMessage(proc process.Process) Exec {
	return Exec{
		proc,
	}
}

func (message Exec) IsMessage() {
}
