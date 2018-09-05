package vm

import (
	"log"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/node"
	"github.com/republicprotocol/smpc-go/core/process"
	"github.com/republicprotocol/smpc-go/core/vm/mul"
	"github.com/republicprotocol/smpc-go/core/vm/open"
	"github.com/republicprotocol/smpc-go/core/vm/rng"
)

const BufferLimit = 1024

type VM struct {
	self  node.Addr
	peers node.Addrs

	nodeBuffer   buffer.Buffer
	nodeSender   buffer.Writer
	nodeReceiver buffer.Reader

	rng             rng.Rnger
	rngBuffer       buffer.Buffer
	rngReaderWriter buffer.ReaderWriter

	mul             mul.Multiplier
	mulBuffer       buffer.Buffer
	mulReaderWriter buffer.ReaderWriter

	open             open.Opener
	openBuffer       buffer.Buffer
	openReaderWriter buffer.ReaderWriter

	internalBuffer       buffer.Buffer
	internalReaderWriter buffer.ReaderWriter
	processes            map[process.ID]process.Process
	processIntents       map[process.ID]process.Intent
}

func New(self node.Addr, peers node.Addrs, nodeSender buffer.Writer, nodeReceiver buffer.Reader, n, k uint, cap int) VM {
	return VM{
		self:  self,
		peers: peers,

		nodeBuffer:   buffer.New(cap),
		nodeSender:   nodeSender,
		nodeReceiver: nodeReceiver,

		rng:             rng.New(),
		rngBuffer:       buffer.New(cap),
		rngReaderWriter: buffer.NewReaderWriter(cap),

		mul:             mul.New(n, k, cap),
		mulBuffer:       buffer.New(cap),
		mulReaderWriter: buffer.NewReaderWriter(cap),

		open:             open.New(n, k, cap),
		openBuffer:       buffer.New(cap),
		openReaderWriter: buffer.NewReaderWriter(cap),

		internalBuffer:       buffer.New(cap),
		internalReaderWriter: buffer.NewReaderWriter(cap),
		processes:            map[process.ID]process.Process{},
		processIntents:       map[process.ID]process.Intent{},
	}
}

func (vm *VM) Run(done <-chan struct{}, reader buffer.Reader, writer buffer.Writer) {
	defer log.Printf("[info] (vm) terminating")

	co.ParBegin(
		func() {
			vm.runBackgroundGoroutines(done)
		},
		func() {
			for {
				select {
				case <-done:
					return

				// Receive messages from an external actor
				case message, ok := <-reader:
					if !ok {
						return
					}
					vm.recvMessage(message)

				// Send messages to a network `node.Node`
				case message := <-vm.nodeBuffer.Peek():
					if !vm.rngBuffer.Pop() {
						log.Printf("[error] (vm) node buffer underflow")
					}
					select {
					case <-done:
					case vm.nodeSender <- message:
					}

				// Receive messages from a network `node.Node`
				case message, ok := <-vm.nodeReceiver:
					if !ok {
						return
					}
					vm.recvMessage(message)

				// Send messages to the `rng.Rnger`
				case message := <-vm.rngBuffer.Peek():
					if !vm.rngBuffer.Pop() {
						log.Printf("[error] (vm) rng buffer underflow")
					}
					select {
					case <-done:
					case vm.rngReaderWriter.Writer() <- message:
					}

				// Send messages to the `mul.Multiplier`
				case message := <-vm.mulBuffer.Peek():
					if !vm.mulBuffer.Pop() {
						log.Printf("[error] (vm) mul buffer underflow")
					}
					select {
					case <-done:
					case vm.mulReaderWriter.Writer() <- message:
					}

				// Send messages to the `open.Opener`
				case message := <-vm.openBuffer.Peek():
					if !vm.rngBuffer.Pop() {
						log.Printf("[error] (vm) open buffer underflow")
					}
					select {
					case <-done:
					case vm.openReaderWriter.Writer() <- message:
					}

				// Send message to an external actor
				case message := <-vm.internalBuffer.Peek():
					if !vm.internalBuffer.Pop() {
						log.Printf("[error] (vm) internal buffer underflow")
					}
					select {
					case <-done:
					case writer <- message:
					}

				// Receive messages from the `rng.Rnger`, `mul.Multiplier`, and
				// the `open.Opener`
				case message := <-vm.internalReaderWriter.Reader():
					vm.recvMessage(message)
				}
			}
		})
}

func (vm *VM) runBackgroundGoroutines(done <-chan struct{}) {
	co.ParBegin(
		func() {
			vm.rng.Run(done, vm.rngReaderWriter.Reader(), vm.internalReaderWriter.Writer())
		},
		func() {
			vm.mul.Run(done, vm.mulReaderWriter.Reader(), vm.internalReaderWriter.Writer())
		},
		func() {
			vm.open.Run(done, vm.openReaderWriter.Reader(), vm.internalReaderWriter.Writer())
		})
}

func (vm *VM) sendMessage(message buffer.Message) {
	if !vm.internalBuffer.Push(message) {
		log.Printf("[error] (vm) buffer overflow")
	}
}

func (vm *VM) sendMessageToNode(message node.Message) {
	if !vm.sender.Push(message) {
		log.Printf("[error] (vm) buffer overflow")
	}
}

func (vm *VM) sendMessageToRng(message buffer.Message) {
	if !vm.rngBuffer.Push(message) {
		log.Printf("[error] (vm) rng buffer overflow")
	}
}

func (vm *VM) sendMessageToMul(message buffer.Message) {
	if !vm.mulBuffer.Push(message) {
		log.Printf("[error] (vm) mul buffer overflow")
	}
}

func (vm *VM) sendMessageToOpen(message buffer.Message) {
	if !vm.openBuffer.Push(message) {
		log.Printf("[error] (vm) open buffer overflow")
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
		vm.sendMessageToRng(rng.NewGenerateRn(rng.Nonce(proc.ID)))

	case process.IntentToMultiply:
		vm.processIntents[proc.ID] = intent
		vm.sendMessageToMul(mul.NewMultiply(mul.Nonce(proc.ID), intent.X, intent.Y, intent.Rho, intent.Sigma))

	case process.IntentToOpen:
		vm.processIntents[proc.ID] = intent
		vm.sendMessageToNode(open.NewOpen(open.Nonce(proc.ID), vm.self, intent.Value))
		vm.sendMessageToOpen(open.NewOpen(open.Nonce(proc.ID), vm.self, intent.Value))

	case process.IntentToError:
		log.Printf("[error] (vm) %v", intent.Error())

	default:
		panic("unimplemented")
	}
}
