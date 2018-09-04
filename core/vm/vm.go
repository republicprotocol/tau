package vm

import (
	"log"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/smpc-go/core/vm/buffer"
	"github.com/republicprotocol/smpc-go/core/vm/mul"
	"github.com/republicprotocol/smpc-go/core/vm/open"
	"github.com/republicprotocol/smpc-go/core/vm/program"
	"github.com/republicprotocol/smpc-go/core/vm/rng"
)

type VM struct {
	buffer   buffer.Buffer
	receiver chan buffer.Message

	rnger       rng.Rnger
	rngerSender chan buffer.Message
	rngerBuffer buffer.Buffer

	multer       mul.Multiplier
	multerSender chan buffer.Message
	multerBuffer buffer.Buffer

	opener       open.Opener
	openerSender chan buffer.Message
	openerBuffer buffer.Buffer

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

		case message := <-vm.rngerBuffer.Peek():
			if !vm.buffer.Pop() {
				log.Printf("[error] (vm) buffer underflow")
			}
			select {
			case <-done:
				return
			case vm.rngerSender <- message:
			}

		case message := <-vm.multerBuffer.Peek():
			if !vm.buffer.Pop() {
				log.Printf("[error] (vm) buffer underflow")
			}
			select {
			case <-done:
				return
			case vm.multerSender <- message:
			}

		case message := <-vm.openerBuffer.Peek():
			if !vm.buffer.Pop() {
				log.Printf("[error] (vm) buffer underflow")
			}
			select {
			case <-done:
				return
			case vm.openerSender <- message:
			}
		}
	}
}

func (vm *VM) runWorkers(done <-chan struct{}) {
	co.ParBegin(
		func() {
			vm.rnger.Run(done, vm.rngerSender, vm.receiver)
		},
		func() {
			vm.multer.Run(done, vm.multerSender, vm.receiver)
		},
		func() {
			vm.opener.Run(done, vm.openerSender, vm.receiver)
		})
}

func (vm *VM) sendMessage(message buffer.Message) {
	if !vm.buffer.Push(message) {
		log.Printf("[error] (vm) buffer overflow")
	}
}

func (vm *VM) sendMessageToRnger(message buffer.Message) {
	if !vm.rngerBuffer.Push(message) {
		log.Printf("[error] (vm) buffer overflow")
	}
}

func (vm *VM) sendMessageToMulter(message buffer.Message) {
	if !vm.multerBuffer.Push(message) {
		log.Printf("[error] (vm) buffer overflow")
	}
}

func (vm *VM) sendMessageToOpener(message buffer.Message) {
	if !vm.openerBuffer.Push(message) {
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

Terminate:
	for {
		ret := prog.Exec()
		vm.progs[prog.ID] = prog

		if ret.IsReady() {
			continue
		}
		if ret.Intent() == nil {
			break
		}

		switch intent := ret.Intent().(type) {
		case program.IntentToGenRn:
			panic("unimplemented")

		case program.IntentToMultiply:
			panic("unimplemented")

		case program.IntentToOpen:
			panic("unimplemented")

		case program.IntentToError:
			log.Printf("[error] (vm) %v", intent.Error())
			break Terminate

		default:
			panic("unimplemented")
		}
	}
}
