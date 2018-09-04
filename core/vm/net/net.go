package net

type Addr uint64

type Networker struct {
}

func (net *Networker) Run(done <-chan struct{}, reader buffer.Reader )
