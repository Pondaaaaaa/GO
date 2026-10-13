package mathx_test

import (
	"testing"

	"github.com/Pondaaaaaa/GO/practices/pz15-tests/internal/mathx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSum_Table(t *testing.T) {
	cases := []struct {
		name       string
		a, b, want int
	}{
		{"positive", 2, 3, 5},
		{"mixed_signs", 10, -5, 5},
		{"both_negative", -4, -6, -10},
		{"both_zero", 0, 0, 0},
		{"zero_identity", 0, -7, -7},
		{"cancellation", -8, 8, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mathx.Sum(tc.a, tc.b); got != tc.want {
				t.Errorf("Sum(%d,%d)=%d; want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestDivide_OkAndError(t *testing.T) {
	cases := []struct {
		name       string
		a, b, want int
		wantErr    bool
	}{
		{"exact", 10, 2, 5, false},
		{"remainder", 7, 3, 2, false},
		{"negative_numerator", -7, 3, -2, false},
		{"negative_denominator", 7, -3, -2, false},
		{"both_negative", -7, -3, 2, false},
		{"zero_numerator", 0, 5, 0, false},
		{"zero_denominator", 10, 0, 0, true},
		{"zero_by_zero", 0, 0, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mathx.Divide(tc.a, tc.b)
			if tc.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, "divide by zero")
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
