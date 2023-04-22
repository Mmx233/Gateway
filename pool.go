package gateway

import "sync"

var BuffPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

type TransBuffPool struct{}

func (a TransBuffPool) Get() []byte {
	return BuffPool.Get().([]byte)
}
func (a TransBuffPool) Put(b []byte) {
	BuffPool.Put(b)
}
