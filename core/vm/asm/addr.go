package asm

type Memory interface {
	Store(offset int, value Value)
	Load(offset int) Value
	Offset(offset int) Memory
}

type Addr struct {
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
	return Addr{mem}
}
func (addr Addr) Store(offset int, value Value) {
	addr.mem[offset] = value
}

func (addr Addr) Load(offset int) Value {
	return addr.mem[offset]
}

func (addr Addr) Offset(offset int) Memory {
	return Addr{addr.mem[offset:]}
}

type AddrIter struct {
	mem  Memory
	step int
}

func NewAddrIter(mem Memory, step int) Memory {
	return AddrIter{mem, step}
}

func (addrIter AddrIter) Store(offset int, value Value) {
	addrIter.mem.Store(offset*addrIter.step, value)
}

func (addrIter AddrIter) Load(offset int) Value {
	return addrIter.mem.Load(offset * addrIter.step)
}

func (addrIter AddrIter) Offset(offset int) Memory {
	return AddrIter{addrIter.mem.Offset(offset * addrIter.step), addrIter.step}
}
