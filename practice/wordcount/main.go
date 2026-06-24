package main

import "fmt"

func main() {
	words := map[string]int{}
	for _, w := range []string{"the", "quick", "brown", "fox", "jumps", "over", "the", "lazy", "dog"} {
		words[w]++
	}
	fmt.Println(words)
}
