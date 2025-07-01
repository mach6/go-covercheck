package formatter_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/formatter"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestFormatAndReport_FailsWhenUnderThreshold(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.StatementThreshold = 80.0
	cfg.BlockThreshold = 70.0
	cfg.SortBy = "file"
	cfg.SortOrder = "asc"

	profiles := []*cover.Profile{
		{
			FileName: "example/foo.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 10, Count: 1},
				{NumStmt: 10, Count: 0},
			},
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	failed := formatter.FormatAndReport(profiles, cfg)

	_ = w.Close()
	os.Stdout = old

	var out bytes.Buffer
	_, _ = out.ReadFrom(r)

	require.True(t, failed)
	require.Contains(t, out.String(), "TOTAL")
}

func TestFormatAndReport_JSONOutput(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatJSON
	cfg.StatementThreshold = 0
	cfg.BlockThreshold = 0

	profiles := []*cover.Profile{
		{
			FileName: "main.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 5, Count: 1},
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	failed := formatter.FormatAndReport(profiles, cfg)

	_ = w.Close()
	os.Stdout = old

	var out bytes.Buffer
	_, _ = out.ReadFrom(r)

	require.False(t, failed)
	require.Contains(t, out.String(), `"file":`)
	require.Contains(t, out.String(), `"main.go"`)
}
