package gateway

import "sync"

var BuffPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}
