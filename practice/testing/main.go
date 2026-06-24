package main

import "fmt"

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	fmt.Println(Max(3, 7))
	fmt.Println(Max(10, 5))
}
