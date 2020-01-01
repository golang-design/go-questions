package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
)

func newBuf() []byte {
	return make([]byte, 10<<20)
}

func main() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	http.HandleFunc("/example2", func(w http.ResponseWriter, r *http.Request) {
		b := newBuf()
		for idx := range b {
			b[idx] = 1
		}
		fmt.Fprintf(w, "done, %v", r.URL.Path[1:])
	})
	http.ListenAndServe(":8080", nil)
}
