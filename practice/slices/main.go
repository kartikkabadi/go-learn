package main

import "fmt"

func main() {
	// Create a slice of strings
	fruits := []string{"apple", "banana", "cherry"}

	// Append "date" to the slice
	fruits = append(fruits, "date")

	// Print each fruit using range
	for _, fruit := range fruits {
		fmt.Println(fruit)
	}

	// Print the length of the slice
	fmt.Println(len(fruits))
}
