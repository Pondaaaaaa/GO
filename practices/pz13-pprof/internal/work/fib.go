package work

// Fib intentionally repeats subproblems to demonstrate a CPU bottleneck.
func Fib(n int) int {
	if n < 2 {
		return n
	}
	return Fib(n-1) + Fib(n-2)
}

// FibFast computes the same value in O(n) time and O(1) space.
func FibFast(n int) int {
	if n < 2 {
		return n
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}
