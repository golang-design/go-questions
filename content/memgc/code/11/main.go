package main

import (
	"os"
	"runtime"
	"runtime/trace"
	"sync/atomic"
)

var stop uint64

// 通过对象 P 的释放状态，来确定 GC 是否已经完成
func gcfinished() *int {
	p := 1
	runtime.SetFinalizer(&p, func(_ *int) {
		println("gc finished")
		atomic.StoreUint64(&stop, 1) // 通知停止分配
	})
	return &p
}

func allocate() {
	// 每次调用分配 0.25MB
	_ = make([]byte, int((1<<20)*0.25))
}

func main() {
	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()

	gcfinished()

	// 当完成 GC 时停止分配
	for n := 1; atomic.LoadUint64(&stop) != 1; n++ {
		println("#allocate: ", n)
		allocate()
	}
	println("terminate")
}
