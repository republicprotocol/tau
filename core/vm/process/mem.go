package process

type Addr = *Value

type Memory = []Value

func NewMemory(cap int) Memory {
	return make(Memory, cap)
}
