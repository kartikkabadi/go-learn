package main

import "fmt"

func main() {
	celsius := 25.0
	fahrenheit := celsius*9.0/5.0 + 32.0
	fmt.Printf("%.0f°C = %.0f°F\n", celsius, fahrenheit)
}
