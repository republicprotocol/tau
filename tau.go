package tau

import "github.com/republicprotocol/tau/core/task"

type (
	IO = task.IO

	Message = task.Message

	MessageID = task.MessageID

	Reducer = task.Reducer

	Task = task.Task
)

var (
	New = task.New

	NewError = task.NewError

	NewIO = task.NewIO

	NewMessageBatch = task.NewMessageBatch

	NewTick = task.NewTick
)
