package vm

import (
	"log"
	"time"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/process"
	"github.com/republicprotocol/smpc-go/core/vm/mul"
	"github.com/republicprotocol/smpc-go/core/vm/open"
	"github.com/republicprotocol/smpc-go/core/vm/rng"
	"github.com/republicprotocol/smpc-go/core/vm/task"
)

type VM struct {
	io             task.IO
	ioExternal     task.IO
	processes      map[process.ID]process.Process
	processIntents map[process.ID]process.Intent

	rng  task.Task
	mul  task.Task
	open task.Task
}

func New(r, w buffer.ReaderWriter, n, k uint, cap int) VM {
	return VM{
		io:             task.NewIO(buffer.New(cap), r.Reader(), w.Writer()),
		ioExternal:     task.NewIO(buffer.New(cap), w.Reader(), r.Writer()),
		processes:      map[process.ID]process.Process{},
		processIntents: map[process.ID]process.Intent{},

		rng:  rng.New(buffer.NewReaderWriter(cap), buffer.NewReaderWriter(cap), time.Minute, nil, nil, n, k, n-k, nil, cap),
		mul:  mul.New(buffer.NewReaderWriter(cap), buffer.NewReaderWriter(cap), n, k, cap),
		open: open.New(buffer.NewReaderWriter(cap), buffer.NewReaderWriter(cap), n, k, cap),
	}
}

func (vm *VM) IO() task.IO {
	return vm.ioExternal
}

func (vm *VM) Run(done <-chan struct{}) {
	defer log.Printf("[info] (vm) terminating")

	co.ParBegin(
		func() {
			vm.runBackgroundGoroutines(done)
		},
		func() {
			for {
				ok := task.Select(
					done,
					vm.recvMessage,
					vm.io,
					vm.rng.IO(),
					vm.mul.IO(),
					vm.open.IO(),
				)
				if !ok {
					return
				}
			}
		})
}

func (vm *VM) runBackgroundGoroutines(done <-chan struct{}) {
	co.ParBegin(
		func() {
			vm.rng.Run(done)
		},
		func() {
			vm.mul.Run(done)
		},
		func() {
			vm.open.Run(done)
		})
}

func (vm *VM) recvMessage(message buffer.Message) {
	switch message := message.(type) {

	case Exec:
		vm.exec(message)

	default:
		log.Printf("[error] (vm) unexpected message type %T", message)
	}
}

func (vm *VM) exec(exec Exec) {
	proc := exec.proc
	vm.processes[proc.ID] = proc

	ret := proc.Exec()
	vm.processes[proc.ID] = proc

	if ret.IsReady() {
		log.Printf("[error] (vm) process is ready after execution = %v", proc.ID)
		return
	}
	if ret.IsTerminated() {
		log.Printf("[debug] (vm) process is terminated = %v", proc.ID)
		return
	}
	if ret.Intent() == nil {
		log.Printf("[debug] (vm) process is waiting = %v", proc.ID)
		return
	}

	switch intent := ret.Intent().(type) {
	case process.IntentToGenerateRn:
		vm.processIntents[proc.ID] = intent
		vm.rng.IO().Send(rng.NewGenerateRn(rng.Nonce(proc.ID)))

	case process.IntentToMultiply:
		vm.processIntents[proc.ID] = intent
		vm.mul.IO().Send(mul.NewMultiply(mul.Nonce(proc.ID), intent.X, intent.Y, intent.Rho, intent.Sigma))

	case process.IntentToOpen:
		vm.processIntents[proc.ID] = intent
		vm.open.IO().Send(open.NewOpen(open.Nonce(proc.ID), intent.Value))

	case process.IntentToError:
		log.Printf("[error] (vm) %v", intent.Error())

	default:
		panic("unimplemented")
	}
}
