package secureconn

import "sync"

const (
	BufSize = 1344
)

var bpool sync.Pool

func init() {
	bpool.New = func() interface{} {
		return make([]byte, BufSize)
	}
}

func BufferPoolGet() []byte {
	return bpool.Get().([]byte)
}
func BufferPoolPut(b []byte) {
	bpool.Put(b)
}
