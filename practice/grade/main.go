package main

import "fmt"

func main() {
	score := 85
	var grade string
	if score >= 90 {
		grade = "A"
	} else if score >= 80 {
		grade = "B"
	} else if score >= 70 {
		grade = "C"
	} else if score >= 60 {
		grade = "D"
	} else {
		grade = "F"
	}
	fmt.Printf("Score %d = Grade %s\n", score, grade)
}
