package main

import "fmt"

// double multiplies the value at the pointer by 2.
func double(v *int) {
	*v = *v * 2
}

func main() {
	x := 21
	double(&x)
	fmt.Println(x)
}
