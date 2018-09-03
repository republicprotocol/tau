package program

type Var interface {
	IsVar()
}

type PrivateVar struct {
}

func (arg PrivateVar) IsVar() {
}

type PublicVar struct {
}

func (arg PublicVar) IsVar() {
}

type Code interface {
	IsCode()
}

type Push struct {
	Argument int
}

func (code Push) IsCode() {
}

type Add struct {
}

func (code Add) IsCode() {
}

type Multiply struct {
}

func (code Multiply) IsCode() {
}

type Open struct {
}

func (code Open) IsCode() {
}
