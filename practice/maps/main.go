package main

import "fmt"

func main() {
	// Create a map with string keys and int values
	ages := map[string]int{"Alice": 30, "Bob": 25}

	// Look up a key using the comma ok idiom
	if age, ok := ages["Alice"]; ok {
		fmt.Println("Alice is", age)
	} else {
		fmt.Println("Alice not found")
	}

	// Print all key-value pairs using range
	for name, age := range ages {
		fmt.Println(name, age)
	}

	// Delete a key and check again
	delete(ages, "Bob")
	if _, ok := ages["Bob"]; ok {
		fmt.Println("Bob is still there")
	} else {
		fmt.Println("Bob was deleted")
	}
}
