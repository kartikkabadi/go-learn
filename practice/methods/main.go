package main

import "fmt"

// Counter tracks an integer value that increments over time.
type Counter struct {
	Value int
}

// Increment adds 1 to the counter using a pointer receiver.
func (c *Counter) Increment() {
	c.Value++
}

func main() {
	c := Counter{}
	c.Increment()
	c.Increment()
	c.Increment()
	fmt.Println(c.Value)
}
