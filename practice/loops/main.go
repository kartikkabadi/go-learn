package main

import "fmt"

func main() {
	// Print numbers 1 to 5 using a for loop
	for i := 1; i <= 5; i++ {
		fmt.Println(i)
	}

	// Print each fruit using range
	fruits := []string{"apple", "banana", "cherry"}
	for _, fruit := range fruits {
		fmt.Println(fruit)
	}
}
