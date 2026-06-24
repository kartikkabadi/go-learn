package main

import "fmt"

func Contains[T comparable](s []T, v T) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

func main() {
	nums := []int{1, 2, 3, 4, 5}
	fmt.Println(Contains(nums, 3))
	fmt.Println(Contains(nums, 10))

	words := []string{"apple", "banana", "cherry"}
	fmt.Println(Contains(words, "banana"))
	fmt.Println(Contains(words, "date"))
}
