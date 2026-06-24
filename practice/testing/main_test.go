package main

import "testing"

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{3, 7, 7},
		{10, 5, 10},
		{-1, -5, -1},
		{0, 0, 0},
		{100, 100, 100},
	}
	for _, tt := range tests {
		got := Max(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("Max(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
