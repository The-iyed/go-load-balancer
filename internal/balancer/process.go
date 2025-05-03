package balancer

import (
	"net/url"
	"sync/atomic"
	"unsafe"
)

type Process struct {
	URL        *url.URL
	Alive      bool
	ErrorCount int32
}

func (p *Process) IsAlive() bool {
	return atomic.LoadUint32((*uint32)(unsafe.Pointer(&p.Alive))) != 0
}

func (p *Process) SetAlive(alive bool) {
	var val uint32
	if alive {
		val = 1
	}
	atomic.StoreUint32((*uint32)(unsafe.Pointer(&p.Alive)), val)
}
