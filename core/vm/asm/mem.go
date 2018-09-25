package asm

type Memory interface {
	Store(offset int, value Value)
	Load(offset int) Value
	Offset(offset int) Memory
}

type memory struct {
	mem []Value
}

// Alloc a contiguous slice of memory and return an addr that points to the
// beginning of the slice. Values can optinally be passed into Alloc. Values
// will be copied into the newly allocated contiguous slice. It will panic if
// there are more Values than the contiguous slice can store.
func Alloc(cap int, values ...Value) Memory {
	mem := make([]Value, cap)
	for i := range values {
		mem[i] = values[i]
	}
	return memory{mem}
}
func (m memory) Store(offset int, value Value) {
	m.mem[offset] = value
}

func (m memory) Load(offset int) Value {
	return m.mem[offset]
}

func (m memory) Offset(offset int) Memory {
	return memory{m.mem[offset:]}
}

type memoryMapper struct {
	mem  Memory
	step int
}

func MemoryMapper(mem Memory, step int) Memory {
	return memoryMapper{mem, step}
}

func (mmapper memoryMapper) Store(offset int, value Value) {
	mmapper.mem.Store(offset*mmapper.step, value)
}

func (mmapper memoryMapper) Load(offset int) Value {
	return mmapper.mem.Load(offset * mmapper.step)
}

func (mmapper memoryMapper) Offset(offset int) Memory {
	return memoryMapper{mmapper.mem.Offset(offset * mmapper.step), mmapper.step}
}
