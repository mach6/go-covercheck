// Package math implements some math helpers.
package math

// PercentFloat returns the percentage of a count and total from float64.
func PercentFloat(count, total float64) float64 {
	if total == 0 {
		return 100.0 //nolint:mnd
	}
	return (count / total) * 100 //nolint:mnd
}

// Percent returns the percentage of a count and total from ints.
func Percent(count, total int) float64 {
	return PercentFloat(float64(count), float64(total))
}
