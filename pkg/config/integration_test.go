package config_test

import (
	"os"
	"path"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

// TestSortByConfigIntegration tests the full integration of sortBy config option
// from YAML file through to the actual sorting of results.
func TestSortByConfigIntegration(t *testing.T) {
	// Create test coverage profiles with different percentages
	profiles := []*cover.Profile{
		{
			FileName: "pkg/high.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 9, Count: 1},  // 9 statements covered
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 1, Count: 0},  // 1 statement not covered
			},
		},
		{
			FileName: "pkg/medium.go", 
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 5, Count: 1},  // 5 statements covered
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 5, Count: 0},  // 5 statements not covered
			},
		},
		{
			FileName: "pkg/low.go",
			Mode:     "set", 
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 1},  // 1 statement covered
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 9, Count: 0},  // 9 statements not covered
			},
		},
	}

	tests := []struct {
		name           string
		yamlContent    string
		expectedOrder  []string  // expected file order
	}{
		{
			name: "sortBy statement-percent desc",
			yamlContent: `
sortBy: statement-percent
sortOrder: desc
`,
			expectedOrder: []string{"high.go", "medium.go", "low.go"}, // 90%, 50%, 10%
		},
		{
			name: "sortBy statement-percent asc", 
			yamlContent: `
sortBy: statement-percent
sortOrder: asc
`,
			expectedOrder: []string{"low.go", "medium.go", "high.go"}, // 10%, 50%, 90%
		},
		{
			name: "sortBy file asc (default)",
			yamlContent: `
sortBy: file
sortOrder: asc
`,
			expectedOrder: []string{"high.go", "low.go", "medium.go"}, // alphabetical
		},
		{
			name: "sortBy file desc",
			yamlContent: `
sortBy: file  
sortOrder: desc
`,
			expectedOrder: []string{"medium.go", "low.go", "high.go"}, // reverse alphabetical
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpFile := path.Join(t.TempDir(), "test_config.yaml")
			err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0600)
			require.NoError(t, err)
			defer os.Remove(tmpFile) //nolint:errcheck

			// Load config
			cfg, err := config.Load(tmpFile)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Collect results using the config
			results, _ := compute.CollectResults(profiles, cfg)

			// Verify the files are sorted according to the config
			require.Len(t, results.ByFile, len(tt.expectedOrder))
			for i, expectedFile := range tt.expectedOrder {
				require.Equal(t, expectedFile, results.ByFile[i].File, 
					"File at position %d should be %s but was %s", i, expectedFile, results.ByFile[i].File)
			}
		})
	}
}