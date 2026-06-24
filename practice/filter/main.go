package main

import "fmt"

func main() {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	var evens []int
	for _, n := range numbers {
		if n%2 == 0 {
			evens = append(evens, n)
		}
	}
	fmt.Println(evens)
}
