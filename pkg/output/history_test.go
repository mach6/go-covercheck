package output_test

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/history"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestCompareHistory(t *testing.T) {
	cPath := test.CreateTempCoverageFile(t, test.TestCoverageOut)
	profiles, err := cover.ParseProfiles(cPath)
	require.NoError(t, err)
	require.NotEmpty(t, profiles)

	results, _ := compute.CollectResults(profiles, new(config.Config))
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		hPath := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
		h, err := history.Load(hPath)
		require.NoError(t, err)
		entry := h.FindByRef("main")
		output.CompareHistory("main", entry, results)
	})

	require.Empty(t, stderr)
	require.Equal(t, `
≡ Comparing against ref: main [commit e402629]
 → By File
    [S] github.com/mach6/go-covercheck/pkg/math/math.go [−25.0%]
    [B] github.com/mach6/go-covercheck/pkg/math/math.go [−25.0%]
    [F] github.com/mach6/go-covercheck/pkg/math/math.go [+100.0%]
 → By Package
    [S] github.com/mach6/go-covercheck/pkg/math [−25.0%]
    [B] github.com/mach6/go-covercheck/pkg/math [−25.0%]
    [F] github.com/mach6/go-covercheck/pkg/math [+100.0%]
 → By Total
    [S] total [+22.2%]
    [B] total [+26.8%]
    [F] total [+100.0%]
`, stdout)
}

func TestShowHistory(t *testing.T) {
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		hPath := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
		h, err := history.Load(hPath)
		require.NoError(t, err)
		output.ShowHistory(h, 1, new(config.Config))
	})
	require.Empty(t, stderr)
	require.Equal(t, `┌────────────┬─────────┬─────────────────┬─────────────────┬─────────────────┬─────────────┐
│  TIMESTAMP │  COMMIT │      BRANCH     │       TAGS      │      LABEL      │   COVERAGE  │
├────────────┼─────────┼─────────────────┼─────────────────┼─────────────────┼─────────────┤
│ 2025-07-18 │ e402629 │ main            │                 │                 │ 180/648 [S] │
│            │         │                 │                 │                 │ 95/409  [B] │
│            │         │                 │                 │                 │         [F] │
└────────────┴─────────┴─────────────────┴─────────────────┴─────────────────┴─────────────┘
≡ Showing last 1 history entry
`, stdout)
}
