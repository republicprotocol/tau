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
	prog := message.prog
	vm.progs[prog.ID] = prog

	for {
		ret := prog.Exec()
		vm.progs[prog.ID] = prog

		if ret.IsReady() {
			continue
		}
		if ret.Intent() == nil {
			break
		}

		switch ret.Intent().(type) {
		case program.IntentToGenRn:
			panic("unimplemented")

		case program.IntentToMultiply:
			panic("unimplemented")

		case program.IntentToOpen:
			panic("unimplemented")

		case program.IntentToError:
			panic("unimplemented")

		default:
			panic("unimplemented")
		}
	}
}
