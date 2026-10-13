package mathx

import "errors"

func Sum(a, b int) int { return a + b }

// Divide performs integer division, truncating toward zero.
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("divide by zero")
	}
	return a / b, nil
}
