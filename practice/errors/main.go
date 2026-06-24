package main

import (
	"errors"
	"fmt"
)

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func main() {
	for _, pair := range [][2]int{{10, 2}, {5, 0}} {
		result, err := divide(pair[0], pair[1])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		} else {
			fmt.Printf("%d / %d = %d\n", pair[0], pair[1], result)
		}
	}
}
