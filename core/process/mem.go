package process

type Addr uint64

type Memory map[Addr]Value

func NewMemory(cap int) Memory {
	return make(Memory, cap)
}
