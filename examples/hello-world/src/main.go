package main

import (
	"fmt"
	"util"
)

func main() {
	fmt.Println("hello world!")

	localIp, err := util.GetLocalIPv4Address()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Local IP: %s\n", localIp)
}