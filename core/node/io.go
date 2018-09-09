package node

import (
	"encoding"

	"github.com/republicprotocol/oro-go/core/buffer"
)

type Sender (chan<- Message)

type Receiver (<-chan Message)

type SenderReceiver (chan Message)

type Message interface {
	buffer.Message
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}
