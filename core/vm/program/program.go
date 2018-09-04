package program

type ID [32]byte

type Addr uint64

type Memory map[Addr]Value

type Program struct {
	ID
	Stack
	Memory
	Code
	PC
}

func New(id ID, stack Stack, mem Memory, code Code) Program {
	return Program{
		ID:     id,
		Stack:  stack,
		Memory: mem,
		Code:   code,
		PC:     0,
	}
}

func (prog *Program) Exec() (bool, error) {
	if prog.PC >= PC(len(prog.Code)) {
		return false, NewExecutionError(ErrCodeOverflow, prog.PC)
	}

	switch inst := prog.Code[prog.PC].(type) {

	case InstPush:
		return prog.execInstPush(inst)

	case InstAdd:
		return prog.execInstAdd(inst)

	default:
		return false, NewUnexpectedInstError(inst, prog.PC)
	}
}

func (prog *Program) execInstPush(inst InstPush) (bool, error) {
	if err := prog.Stack.Push(inst.Value); err != nil {
		return false, err
	}
	prog.PC++

	return true, nil
}

func (prog *Program) execInstAdd(inst InstAdd) (bool, error) {
	rhs, err := prog.Stack.Pop()
	if err != nil {
		return false, err
	}
	lhs, err := prog.Stack.Pop()
	if err != nil {
		return false, err
	}

	ret := lhs.Add(rhs)
	if err := prog.Stack.Push(ret); err != nil {
		return false, err
	}
	prog.PC++

	return true, nil
}
