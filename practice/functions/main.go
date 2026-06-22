package main

import "fmt"

// double returns twice the input number.
func double(x int) int {
	return x * 2
}

// greet returns a greeting message using the given name.
func greet(name string) string {
	return "Hello, " + name + "!"
}

func main() {
	fmt.Println(double(5))
	fmt.Println(greet("Go learner"))
}
