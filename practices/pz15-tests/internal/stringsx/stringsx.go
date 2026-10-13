package stringsx

// Clip limits the number of bytes, not runes. It may split a UTF-8 sequence.
// Negative max is treated as zero.
func Clip(s string, max int) string {
	if max < 0 {
		max = 0
	}
	if len(s) <= max {
		return s
	}
	return s[:max]
}
