package history

import (
	"os"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
	h, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, h)
	require.Len(t, h.Entries, 1)
}

func TestNew(t *testing.T) {
	h := New("")
	require.NotNil(t, h)
	require.Equal(t, "", h.path)
}

func TestHistory_AddResults(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	h.AddResults(compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{
				Coverage:   "1/1",
				Threshold:  0,
				Percentage: 0,
				Failed:     false,
			},
			Blocks: compute.TotalBlocks{
				Coverage:   "2/2",
				Threshold:  0,
				Percentage: 0,
				Failed:     false,
			},
		},
	}, "testing")

	require.Len(t, h.Entries, 1)
	require.Equal(t, "unknown", h.Entries[0].Commit)
	require.Equal(t, "unknown", h.Entries[0].Branch)
}

func TestHistory_FindByRef(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
	h, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, h)

	entry := h.FindByRef("main")
	require.NotNil(t, entry)
	require.Equal(t, "e40262964cc463a18753e2834c04230c2a356f20", entry.Commit)
	require.Equal(t, "main", entry.Branch)
}

func TestHistory_Save(t *testing.T) {
	path := t.TempDir() + "/history"
	defer t.Cleanup(func() {
		_ = os.RemoveAll(t.TempDir())
	})
	h := New(path)
	require.NotNil(t, h)
	err := h.Save(0)
	require.NoError(t, err)
}
