package process

type Addr = *Value

type Memory []Value

func NewMemory(cap int) Memory {
	return make(Memory, cap)
}

func (mem Memory) At(i int) *Value {
	return &mem[i]
}
