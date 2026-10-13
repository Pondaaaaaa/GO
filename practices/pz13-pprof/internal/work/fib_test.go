package work

import "testing"

// Sink keeps the result observable outside the benchmark loop.
var Sink int
var benchmarkN = 30

func TestFib(t *testing.T) {
	for _, tt := range []struct{ n, want int }{{0, 0}, {1, 1}, {2, 1}, {10, 55}, {30, 832040}} {
		if got := Fib(tt.n); got != tt.want {
			t.Errorf("Fib(%d)=%d; want %d", tt.n, got, tt.want)
		}
		if got := FibFast(tt.n); got != tt.want {
			t.Errorf("FibFast(%d)=%d; want %d", tt.n, got, tt.want)
		}
	}
}

func BenchmarkFib(b *testing.B) {
	n := benchmarkN
	if Fib(n) != 832040 {
		b.Fatal("wrong result")
	}
	b.ReportAllocs()
	b.ResetTimer()
	var result int
	for i := 0; i < b.N; i++ {
		result = Fib(n)
	}
	b.StopTimer()
	Sink = result
	if result != 832040 {
		b.Fatal("wrong benchmark result")
	}
}

func BenchmarkFibFast(b *testing.B) {
	n := benchmarkN
	if FibFast(n) != 832040 {
		b.Fatal("wrong result")
	}
	b.ReportAllocs()
	b.ResetTimer()
	var result int
	for i := 0; i < b.N; i++ {
		result = FibFast(n)
	}
	b.StopTimer()
	Sink = result
	if result != 832040 {
		b.Fatal("wrong benchmark result")
	}
}
