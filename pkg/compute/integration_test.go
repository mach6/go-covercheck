package compute

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"golang.org/x/tools/cover"
)

func TestCollectResults_WithUncoveredLines(t *testing.T) {
	profiles := []*cover.Profile{
		{
			FileName: "test.go",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, EndLine: 1, NumStmt: 1, Count: 1},
				{StartLine: 2, EndLine: 3, NumStmt: 2, Count: 0},
				{StartLine: 4, EndLine: 4, NumStmt: 1, Count: 1},
			},
		},
	}

	t.Run("with uncovered lines enabled", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()

		results, _ := CollectResults(profiles, cfg)

		if len(results.ByFile) != 1 {
			t.Fatalf("expected 1 file result, got %d", len(results.ByFile))
		}

		fileResult := results.ByFile[0]
		if fileResult.UncoveredLines != "2-3" {
			t.Errorf("expected uncovered lines '2-3', got '%s'", fileResult.UncoveredLines)
		}
	})

	t.Run("with uncovered lines disabled", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.HideUncoveredLines = true

		results, _ := CollectResults(profiles, cfg)

		if len(results.ByFile) != 1 {
			t.Fatalf("expected 1 file result, got %d", len(results.ByFile))
		}

		fileResult := results.ByFile[0]
		if fileResult.UncoveredLines != "" {
			t.Errorf("expected empty uncovered lines, got '%s'", fileResult.UncoveredLines)
		}
	})
}
