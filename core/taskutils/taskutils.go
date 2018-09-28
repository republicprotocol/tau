package taskutils

import (
	"fmt"
	"math/rand"

	"github.com/republicprotocol/tau/core/task"
)

// RouteMessage to multiple tasks. A simulated failure rate in the range [0.0,
// 1.0] defines the probability that a message will be dropped for any one task.
// A simulated failure limit is an absolute limit on the number of simulated
// failures that can happen.
func RouteMessage(done <-chan struct{}, msg task.Message, ts task.Tasks, simulatedFailureRate float64, simulatedFailureLimit int) (simulatedFailures int) {
	for _, t := range ts {
		if simulatedFailures < simulatedFailureLimit && rand.Float64() < simulatedFailureRate {
			simulatedFailures++
			continue
		}
		t.IO().InputWriter() <- msg
	}
	return
}

// RandomMessageID returns a random message ID. This function panics if an error
// prevents it from generating the entire message ID.
func RandomMessageID() task.MessageID {
	msgid := task.MessageID{}
	n, err := rand.Read(msgid[:])
	if n != len(msgid) {
		if err != nil {
			panic(fmt.Sprintf("failed to generate %v random bytes = %v", n, err))
		}
		panic(fmt.Sprintf("failed to generate %v random bytes", n))
	}
	if err != nil {
		panic(err)
	}
	return msgid
}
