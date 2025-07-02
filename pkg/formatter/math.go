package formatter

func percentFloat(count, total float64) float64 {
	if total == 0 {
		return 100.0 //nolint:mnd
	}
	return (count / total) * 100 //nolint:mnd
}

func percent(count, total int) float64 {
	return percentFloat(float64(count), float64(total))
}
