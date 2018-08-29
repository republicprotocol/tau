package mul

import "time"

type Multiplier interface {
	Run(done <-chan (struct{}), input <-chan InputMessage, output chan<- OutputMessage)
}

type multiplier struct {
	timeout               time.Duration
	addr, leader, n, k, t uint

	sendBuffer    []OutputMessage
	sendBufferCap int
}

func (multer *multiplier) Run(done <-chan (struct{}), input <-chan InputMessage, output chan<- OutputMessage) {
	for {
		var outputMessage OutputMessage
		var outputMaybe chan<- OutputMessage
		if len(multer.sendBuffer) > 0 {
			outputMessage = multer.sendBuffer[0]
			outputMaybe = output
		}

		select {
		case <-done:
			return

		case message, ok := <-input:
			if !ok {
				return
			}
			multer.recvMessage(message)

		case outputMaybe <- outputMessage:
			multer.sendBuffer = multer.sendBuffer[1:]
		}
	}
}

func (multer *multiplier) isLeader() bool {
	return multer.leader == multer.addr
}

func (multer *multiplier) sendMessage(message OutputMessage) {
	multer.sendBuffer = append(multer.sendBuffer, message)
}

func (multer *multiplier) recvMessage(message InputMessage) {
	switch message := message.(type) {
	case Nominate:
		// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		multer.nominate(message)

	case Mul:
		// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		multer.multiply(message)
	}
}

func (multer *multiplier) nominate(message Nominate) {
	multer.leader = message.Leader
}

func (multer *multiplier) multiply(message Mul) {
}
