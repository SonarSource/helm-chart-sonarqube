package calculator

import (
	"testing"
)

func TestCalculator_Add(t *testing.T) {
	calc := New()
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 5, 3, 8},
		{"negative numbers", -2, -3, -5},
		{"zero", 0, 5, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.Add(tt.a, tt.b); got != tt.expected {
				t.Errorf("Add() = %v, want %v", got, tt.expected)
			}
		})
	}
}
func TestCalculator_Multiply(t *testing.T) {
	calc := New()
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 4, 3, 12},
		{"negative numbers", -2, 3, -6},
		{"zero", 0, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.Multiply(tt.a, tt.b); got != tt.expected {
				t.Errorf("Multiply() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculator_Divide(t *testing.T) {
	calc := New()
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 10, 2, 5},
		{"negative numbers", -10, 2, -5},
		{"zero", 0, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.Divide(tt.a, tt.b); got != tt.expected {
				t.Errorf("Divide() = %v, want %v", got, tt.expected)
			}
		})
	}
}
