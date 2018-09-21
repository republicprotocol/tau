package asm

// An Addr points a location in memory where Values can be stored.
type Addr struct {
	mem []Value
}

// Alloc a contiguous slice of memory and return an Addr that points to the
// beginning of the slice. Values can optinally be passed into Alloc. Values
// will be copied into the newly allocated contiguous slice. It will panic if
// there are more Values than the contiguous slice can store.
func Alloc(cap int, values ...Value) Addr {
	addr := Addr{
		mem: make([]Value, cap),
	}
	for i := range values {
		addr.mem[i] = values[i]
	}
	return addr
}

// Store a Value at an offset from the Addr. It will panic if the offset extends
// beyond the range of allocated memory.
func (addr Addr) Store(offset int, value Value) {
	addr.mem[offset] = value
}

// Load a Value from an offset from the Addr. It will panic if the offset
// extends beyond the range of allocated memory.
func (addr Addr) Load(offset int) Value {
	return addr.mem[offset]
}

// Offset returns an Addr at an offset from another Addr. It will panic if the
// offset extends beyond the range of allocated memory.
func (addr Addr) Offset(offset int) Addr {
	if offset >= len(addr.mem) {
		panic("offset out of address space")
	}
	return Addr{
		mem: addr.mem[offset:],
	}
}

type AddrIter struct {
	addr Addr
	step int
}

func NewAddrIter(addr Addr, step int) AddrIter {
	return AddrIter{addr, step}
}

func (addrIter AddrIter) Store(offset int, value Value) {
	addrIter.addr.Store(offset*addrIter.step, value)
}

func (addrIter AddrIter) Load(offset int) Value {
	return addrIter.addr.Load(offset * addrIter.step)
}

func (addrIter AddrIter) Offset(offset int) AddrIter {
	return AddrIter{addrIter.addr.Offset(offset * addrIter.step), addrIter.step}
}
