package vm

import (
	"log"

	"github.com/republicprotocol/co-go"

	"github.com/republicprotocol/smpc-go/core/vm/buffer"
	"github.com/republicprotocol/smpc-go/core/vm/mul"
	"github.com/republicprotocol/smpc-go/core/vm/program"
	"github.com/republicprotocol/smpc-go/core/vm/rng"
)

type VM struct {
	buffer   buffer.Buffer
	receiver chan buffer.Message

	rng       rng.Rnger
	rngSender chan buffer.Message
	rngBuffer buffer.Buffer

	mul       mul.Multiplier
	mulSender chan buffer.Message
	mulBuffer buffer.Buffer

	progs map[program.ID]program.Program
}

func (vm *VM) Run(done <-chan struct{}, sender <-chan buffer.Message, receiver chan<- buffer.Message) {
	defer log.Printf("[info] (vm) terminating")

	go vm.runWorkers(done)

	for {
		select {
		case <-done:
			return

		case message, ok := <-sender:
			if !ok {
				return
			}
			vm.recvMessage(message)

		case message, ok := <-vm.receiver:
			if !ok {
				return
			}
			vm.recvMessage(message)

		case message := <-vm.buffer.Peek():
			if !vm.buffer.Pop() {
				log.Printf("[error] (vm) buffer underflow")
			}
			select {
			case <-done:
				return
			case receiver <- message:
			}

		case message := <-vm.rngBuffer.Peek():
			if !vm.buffer.Pop() {
				log.Printf("[error] (vm) buffer underflow")
			}
			select {
			case <-done:
				return
			case receiver <- message:
			}

		case message := <-vm.mulBuffer.Peek():
			if !vm.buffer.Pop() {
				log.Printf("[error] (vm) buffer underflow")
			}
			select {
			case <-done:
				return
			case receiver <- message:
			}
		}
	}
}

func (vm *VM) runWorkers(done <-chan struct{}) {
	co.ParBegin(
		func() {
			vm.rng.Run(done, vm.rngSender, vm.receiver)
		},
		func() {
			vm.mul.Run(done, vm.mulSender, vm.receiver)
		})
}

func (vm *VM) sendMessage(message buffer.Message) {
	if !vm.buffer.Push(message) {
		log.Printf("[error] (vm) buffer overflow")
	}
}

func (vm *VM) sendMessageToRng(message buffer.Message) {
	if !vm.rngBuffer.Push(message) {
		log.Printf("[error] (vm) buffer overflow")
	}
}

func (vm *VM) sendMessageToMul(message buffer.Message) {
	if !vm.mulBuffer.Push(message) {
		log.Printf("[error] (vm) buffer overflow")
	}
}

func (vm *VM) recvMessage(message buffer.Message) {
	switch message := message.(type) {

	case Exec:
		vm.exec(message)

	default:
		log.Printf("[error] (vm) unexpected message type %T", message)
	}
}

func (vm *VM) exec(message Exec) {
	vm.progs[message.prog.ID] = message.prog

	for vm.execStep(vm.progs[message.prog.ID]) {
	}
}

func (vm *VM) execStep(prog program.Program) bool {
	defer func() {
		vm.progs[prog.ID] = prog
	}()

	switch code := prog.Code[prog.PC].(type) {
	case program.Push:
		return vm.execPush(&prog, code)
	case program.Add:
		return vm.execAdd(&prog, code)
	case program.Multiply:
		return vm.execMul(&prog, code)
	case program.Open:
		return vm.execOpen(&prog, code)
	default:
		log.Printf("[error] (vm) unexpected code type %T", code)
		return true
	}
}

func (vm *VM) execPush(prog *program.Program, push program.Push) bool {
	return true
}

func (vm *VM) execAdd(prog *program.Program, add program.Add) bool {
	return true
}

func (vm *VM) execMul(prog *program.Program, mul program.Multiply) bool {
	return true
}

func (vm *VM) execOpen(prog *program.Program, open program.Open) bool {
	return true
}
