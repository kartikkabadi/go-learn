package main

import (
	"fmt"
	"math"
)

type Shape interface {
	Area() float64
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

type Rectangle struct {
	Width  float64
	Height float64
}

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func main() {
	c := Circle{Radius: 5}
	r := Rectangle{Width: 3, Height: 4}
	fmt.Printf("Circle: %.2f\n", c.Area())
	fmt.Printf("Rectangle: %.2f\n", r.Area())
}
