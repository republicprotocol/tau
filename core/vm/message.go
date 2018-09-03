package vm

import (
	"github.com/republicprotocol/smpc-go/core/vm/program"
)

type Exec struct {
	prog program.Program
}

func NewExecMessage(prog program.Program) Exec {
	return Exec{
		prog,
	}
}

func (message Exec) IsMessage() {
}
