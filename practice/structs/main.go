package main

import "fmt"

// Person represents a person with a name and age.
type Person struct {
	Name string
	Age  int
}

func main() {
	// Create a Person value using a struct literal
	p1 := Person{Name: "Alice", Age: 30}

	// Access and print the fields
	fmt.Println(p1.Name)
	fmt.Println(p1.Age)

	// Create another Person and modify a field
	p2 := Person{Name: "Bob", Age: 25}
	p2.Age = 26

	fmt.Println(p2.Name)
	fmt.Println(p2.Age)
}
