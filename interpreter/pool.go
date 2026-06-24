package interpreter

import (
	"sync"

	"github.com/cryptoddev/fastvm/memory"
)

var framePool = sync.Pool{
	New: func() interface{} {
		return &Frame{Memory: memory.New()}
	},
}

func AcquireFrame() *Frame {
	f := framePool.Get().(*Frame)
	f.Reset()
	return f
}

func ReleaseFrame(f *Frame) {
	if cap(f.Code) > 65536 || cap(f.Input) > 65536 {
		return
	}
	f.Code = nil
	f.Input = nil
	f.ReturnData = f.ReturnData[:0]
	f.State = nil
	f.Block = nil
	f.Tx = nil
	framePool.Put(f)
}
