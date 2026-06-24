package main

import (
	"fmt"
	"sort"
	"strings"
)

func main() {
	input := "cherry,apple,banana,date"
	parts := strings.Split(input, ",")
	sort.Strings(parts)
	result := strings.Join(parts, ",")
	fmt.Println(result)
}
