package main_test

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func BenchmarkGCLargeGs(b *testing.B) {
	wg := sync.WaitGroup{}

	for ng := 100; ng <= 1000000; ng *= 10 {
		b.Run(fmt.Sprintf("#g-%d", ng), func(b *testing.B) {
			// Prepare loads of goroutines and wait
			// all goroutines terminate.
			wg.Add(ng)
			for i := 0; i < ng; i++ {
				go func() {
					time.Sleep(100 * time.Millisecond)
					wg.Done()
				}()
			}
			wg.Wait()

			// Run GC once for cleanup
			runtime.GC()

			// Now record GC scalability
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				runtime.GC()
			}
		})

	}
}
