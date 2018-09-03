package program

type ID [32]byte

type Program struct {
	ID

	Args  []Var
	Stack []Var

	Code []Code
	PC   int
}

func New(id ID, args []Var, code []Code) Program {
	return Program{
		ID:    id,
		Args:  args,
		Stack: make([]Var, 0, len(args)),
		Code:  code,
		PC:    0,
	}
}
