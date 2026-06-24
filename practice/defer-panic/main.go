package main

import "fmt"

func main() {
	fmt.Println("start")
	defer fmt.Println("first defer")
	defer fmt.Println("second defer")
	fmt.Println("end")
}
