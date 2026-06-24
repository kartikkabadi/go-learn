package main

import (
	"fmt"
	"os"
)

func main() {
	err := os.WriteFile("hello.txt", []byte("Hello, file!"), 0644)
	if err != nil {
		fmt.Println("Write error:", err)
		return
	}

	data, err := os.ReadFile("hello.txt")
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}
	fmt.Println(string(data))
}
