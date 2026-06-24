package main

import (
	"fmt"
	"time"
)

func printNum(n int) {
	fmt.Println(n)
}

func main() {
	for i := 1; i <= 5; i++ {
		go printNum(i)
	}
	time.Sleep(100 * time.Millisecond)
}
