package program

type PC uint64

type Code []Inst

type Inst interface {
	IsInst()
}

type InstPush struct {
	Value
}

func (inst InstPush) IsInst() {
}

type InstAdd struct {
}

func (inst InstAdd) IsInst() {
}
