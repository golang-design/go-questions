package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/trace"
	"time"
)

const (
	windowSize = 200000
	msgCount   = 1000000
)

var (
	best    time.Duration = time.Second
	bestAt  time.Time
	worst   time.Duration
	worstAt time.Time

	start = time.Now()
)

func main() {
	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()

	for i := 0; i < 5; i++ {
		measure()
		worst = 0
		best = time.Second
		runtime.GC()
	}
}

func measure() {
	var c channel
	for i := 0; i < msgCount; i++ {
		c.sendMsg(i)
	}
	fmt.Printf("Best send delay %v at %v, worst send delay: %v at %v. Wall clock: %v \n", best, bestAt.Sub(start), worst, worstAt.Sub(start), time.Since(start))
}

type channel [windowSize][]byte

func (c *channel) sendMsg(id int) {
	start := time.Now()

	// 模拟发送
	(*c)[id%windowSize] = newMsg(id)

	end := time.Now()
	elapsed := end.Sub(start)
	if elapsed > worst {
		worst = elapsed
		worstAt = end
	}
	if elapsed < best {
		best = elapsed
		bestAt = end
	}
}

func newMsg(n int) []byte {
	m := make([]byte, 1024)
	for i := range m {
		m[i] = byte(n)
	}
	return m
}
