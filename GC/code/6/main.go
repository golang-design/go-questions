package main

import (
	"os"
	"runtime/trace"
)

func main() {
	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()
	for i := 0; i < 100000; i++ {
		go func() {
			select {}
		}()
	}
}
