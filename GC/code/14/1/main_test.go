package main_test

import (
	"fmt"
	"runtime"
	"testing"
)

func BenchmarkExample1(b *testing.B) {
	for j := 1; j < 20; j++ {
		b.Run(fmt.Sprintf("%d", j), func(b *testing.B) {
			b.ReportAllocs()
			a := make([]byte, 10<<20)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				runtime.GC()
			}

			runtime.KeepAlive(a)
		})
	}
}
