package vm

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vm/asm"
	"github.com/republicprotocol/oro-go/core/vm/proc"
)

type Exec struct {
	process proc.Proc
}

func NewExec(process proc.Proc) Exec {
	return Exec{
		process,
	}
}

func (message Exec) String() string {
	return fmt.Sprintf("vm.Exec {\n\tproc: %v\n}", message.process)
}

func (message Exec) IsMessage() {
}

type Result struct {
	Values []asm.Value
}

func (message Result) String() string {
	return fmt.Sprintf("vm.Result {\n\tvalues: %v\n}", message.Values)
}

func NewResult(values []asm.Value) Result {
	return Result{values}
}

func (message Result) IsMessage() {
}

type RemoteProcedureCall struct {
	Message task.Message
}

func NewRemoteProcedureCall(message task.Message) RemoteProcedureCall {
	return RemoteProcedureCall{
		message,
	}
}

func (message RemoteProcedureCall) IsMessage() {
}

func (message RemoteProcedureCall) String() (ret string) {
	// switch msg := message.Message.(type) {
	// case rng.Nominate:
	// 	ret = msg.String()
	// case rng.GenerateRn:
	// 	ret = msg.String()
	// case rng.ProposeRn:
	// 	ret = msg.String()
	// case rng.LocalRnShares:
	// 	ret = msg.String()
	// case rng.ProposeGlobalRnShare:
	// 	ret = msg.String()
	// case rng.GlobalRnShare:
	// 	ret = msg.String()
	// case rng.VoteGlobalRnShare:
	// 	ret = msg.String()
	// case rng.CheckDeadline:
	// 	ret = msg.String()
	// case rng.Err:
	// 	ret = msg.String()

	// case open.Open:
	// 	ret = msg.String()
	// case open.BroadcastShare:
	// 	ret = msg.String()
	// case open.Result:
	// 	ret = msg.String()

	// case mul.Mul:
	// 	ret = msg.String()
	// case mul.BroadcastIntermediateShare:
	// 	ret = msg.String()
	// case mul.Result:
	// 	ret = msg.String()
	// }
	// return ret
	return "rpc"
}
