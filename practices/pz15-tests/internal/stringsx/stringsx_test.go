package stringsx_test

import (
	"testing"
	"unicode/utf8"

	"github.com/Pondaaaaaa/GO/practices/pz15-tests/internal/stringsx"
	"github.com/stretchr/testify/assert"
)

func TestClip(t *testing.T) {
	cases := []struct {
		name, input string
		max         int
		want        string
	}{
		{"empty", "", 5, ""},
		{"empty_zero", "", 0, ""},
		{"empty_negative", "", -1, ""},
		{"zero", "hello", 0, ""},
		{"negative", "hello", -2, ""},
		{"equal_length", "hello", 5, "hello"},
		{"larger_limit", "hello", 10, "hello"},
		{"truncate", "hello", 3, "hel"},
		{"first_byte", "hello", 1, "h"},
		{"unicode_whole_rune", "Привет", 2, "П"},
		{"unicode_split", "П", 1, "\xd0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, stringsx.Clip(tc.input, tc.max))
		})
	}
	t.Run("split_utf8_is_documented", func(t *testing.T) {
		got := stringsx.Clip("П", 1)
		assert.Equal(t, []byte{0xd0}, []byte(got))
		assert.False(t, utf8.ValidString(got), "byte clipping can produce invalid UTF-8")
	})
}
