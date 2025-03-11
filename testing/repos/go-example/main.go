package main

import (
	"fmt"
	"go-example/calculator"
)

func main() {
	calc := calculator.New()
	result := calc.Add(10, 5)
	fmt.Printf("10 + 5 = %d\n", result)
	result = calc.Multiply(6, 4)
	fmt.Printf("6 4 = %d\n", result)
}
