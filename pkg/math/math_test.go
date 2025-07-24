package math_test

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/math"
	"github.com/stretchr/testify/require"
)

func TestPercent(t *testing.T) {
	expect := 50.0
	actual := math.Percent(50, 100)
	require.InEpsilon(t, expect, actual, 0)
}

func TestPercentFloat(t *testing.T) {
	expect := 50.0
	actual := math.PercentFloat(50.0, 100)
	require.InEpsilon(t, expect, actual, 0)

	expect = 100
	actual = math.PercentFloat(100, 0)
	require.InEpsilon(t, expect, actual, 0)
}
