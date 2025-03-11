package calculator

// Calculator represents a basic calculator
type Calculator struct{}

// New creates a new Calculator instance
func New() *Calculator {
	return &Calculator{}
}

// Add returns the sum of two integers
func (c Calculator) Add(a, b int) int {
	return a + b
}

// Subtract returns the difference between two integers
func (c Calculator) Subtract(a, b int) int {
	return a - b
}

// Multiply returns the product of two integers
func (c Calculator) Multiply(a, b int) int {
	return a * b
}

// Divide returns the quotient of two integers
// Panics if divisor is zero
func (c Calculator) Divide(a, b int) int {
	if b == 0 {
		panic("division by zero")
	}
	return a / b
}
