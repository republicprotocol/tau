package proc

import (
	"github.com/republicprotocol/oro-go/core/vm/asm"
)

type IntentID [40]byte

type Intent struct {
	iid   IntentID
	state asm.State
}

func NewIntent(iid IntentID, state asm.State) Intent {
	return Intent{
		iid:   iid,
		state: state,
	}
}

func (intent Intent) IID() IntentID {
	return intent.iid
}

func (intent Intent) State() asm.State {
	return intent.state
}
