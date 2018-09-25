package node

import (
	"log"

	"github.com/republicprotocol/oro-go/core/collection/buffer"
)

// A Node in the network can send and receive messages to and from other Nodes
// over a network interface.
type Node interface {
	Run(done <-chan struct{}, receiver Receiver, sender Sender)
}

type node struct {
	buffer buffer.Buffer
}

func (node *node) Run(done <-chan struct{}, receiver Receiver, sender Sender) {
	defer log.Printf("[info] (node) terminating")

	for {
		select {
		case <-done:
			return

		case message, ok := <-receiver:
			if !ok {
				return
			}
			node.recvMessage(message)

		case message, ok := <-node.buffer.Peek():
			if !ok {
				return
			}
			select {
			case <-done:
				return
			case sender <- message.(Message):
			}
		}
	}
}

func (node *node) recvMessage(message Message) {
	switch message := message.(type) {
	default:
		log.Printf("[error] (node) unexpected message type %T", message)
	}
}
