package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/trace"
	"sync/atomic"
	"time"
)

var (
	stop  int32
	count int64
	sum   time.Duration
)

func concat() {
	for n := 0; n < 100; n++ {
		for i := 0; i < 8; i++ {
			go func() {
				s := "Go GC"
				s += " " + "Hello"
				s += " " + "World"
				_ = s
			}()
		}
	}
}

func main() {
	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()

	go func() {
		var t time.Time
		for atomic.LoadInt32(&stop) == 0 {
			t = time.Now()
			runtime.GC()
			sum += time.Since(t)
			count++
		}
		fmt.Printf("GC spend avg: %v\n", time.Duration(int64(sum)/count))
	}()

	concat()
	atomic.StoreInt32(&stop, 1)
}
