package proc

import (
	"encoding/base64"

	"github.com/republicprotocol/oro-go/core/vm/asm"
)

type PC uint64

type ID [32]byte

func (id ID) String() string {
	idBase64 := base64.StdEncoding.EncodeToString(id[:])
	idRunes := []rune(idBase64)
	return string(idRunes[16:])
}

type Proc struct {
	ID

	PC
	Insts  []asm.Inst
	States []asm.State
}

func New(id ID, insts []asm.Inst) Proc {
	expandMacros(&insts)
	return Proc{
		ID: id,

		PC:     0,
		Insts:  insts,
		States: make([]asm.State, len(insts)),
	}
}

func (process *Proc) Exec() Intent {
	for {

		// Execute the instruction and store the resulting state
		result := process.Insts[process.PC].Eval(process.States[process.PC])
		process.States[process.PC] = result.State

		switch result.State.(type) {
		case *asm.InstExitState:
			// If the state is an exit state then return an exit intent
			return NewIntent(process.iid(), result.State)
		default:
			// Otherwise if the state is not ready then return a transition
			// intent
			if !result.Ready {
				return NewIntent(process.iid(), result.State)
			}
		}

		process.PC++
	}
}

func (process *Proc) iid() IntentID {
	id := IntentID{}
	copy(id[:32], process.ID[:32])
	id[32] = byte(process.PC)
	id[33] = byte(process.PC >> 8)
	id[34] = byte(process.PC >> 16)
	id[35] = byte(process.PC >> 24)
	id[36] = byte(process.PC >> 32)
	id[37] = byte(process.PC >> 40)
	id[38] = byte(process.PC >> 48)
	id[39] = byte(process.PC >> 56)
	return id
}

func expandMacros(code *[]asm.Inst) {
	for i := 0; i < len(*code); i++ {
		insts, instDidExpand := (*code)[i].Expand()
		if instDidExpand {
			temp := append(insts, (*code)[i+1:]...)
			*code = append((*code)[:i], temp...)
			i--
		}
	}
}
