package vm

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/buffer"
	"github.com/republicprotocol/oro-go/core/vm/mul"
	"github.com/republicprotocol/oro-go/core/vm/open"
	"github.com/republicprotocol/oro-go/core/vm/process"
	"github.com/republicprotocol/oro-go/core/vm/rng"
)

type Exec struct {
	proc process.Process
}

func NewExec(proc process.Process) Exec {
	return Exec{
		proc,
	}
}

func (message Exec) String() string {
	return fmt.Sprintf("vm.Exec {\n\tproc: %v\n}", message.proc)
}

func (message Exec) IsMessage() {
}

type Result struct {
	Value process.Value
}

func (message Result) String() string {
	return fmt.Sprintf("vm.Result {\n\tvalue: %v\n}", message.Value)
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

func (message RemoteProcedureCall) String() (ret string) {
	switch msg := message.Message.(type) {
	case rng.Nominate:
		ret = msg.String()
	case rng.GenerateRn:
		ret = msg.String()
	case rng.ProposeRn:
		ret = msg.String()
	case rng.LocalRnShares:
		ret = msg.String()
	case rng.ProposeGlobalRnShare:
		ret = msg.String()
	case rng.GlobalRnShare:
		ret = msg.String()
	case rng.VoteGlobalRnShare:
		ret = msg.String()
	case rng.CheckDeadline:
		ret = msg.String()
	case rng.Err:
		ret = msg.String()

	case open.Open:
		ret = msg.String()
	case open.BroadcastShare:
		ret = msg.String()
	case open.Result:
		ret = msg.String()

	case mul.Mul:
		ret = msg.String()
	case mul.BroadcastIntermediateShare:
		ret = msg.String()
	case mul.Result:
		ret = msg.String()
	}
	return ret
}
