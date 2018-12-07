package tau

import "github.com/republicprotocol/tau/core/task"

type (
	Error = task.Error

	IO = task.IO

	Message = task.Message

	MessageID = task.MessageID

	Reducer = task.Reducer

	ReduceFunc = task.ReduceFunc

	Task = task.Task

	Tasks = task.Tasks

	Tick = task.Tick
)

var (
	New = task.New

	NewError = task.NewError

	NewIO = task.NewIO

	NewMessageBatch = task.NewMessageBatch

	NewTick = task.NewTick
)
