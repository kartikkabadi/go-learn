package main

import (
	"encoding/json"
	"fmt"
)

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func main() {
	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99},
		{ID: 2, Name: "Mouse", Price: 29.99},
	}

	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		fmt.Println("JSON error:", err)
		return
	}
	fmt.Println(string(data))
}
