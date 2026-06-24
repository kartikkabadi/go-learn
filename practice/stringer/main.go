package main

import "fmt"

type Person struct {
	Name string
	Age  int
}

func (p Person) String() string {
	return fmt.Sprintf("%s (%d)", p.Name, p.Age)
}

func main() {
	p := Person{Name: "Alice", Age: 30}
	fmt.Println(p)
}
