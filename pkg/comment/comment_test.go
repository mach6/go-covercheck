package comment

import (
	"strings"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

func TestFormatMarkdown(t *testing.T) {
	// Create test results
	results := compute.Results{
		ByFile: []compute.ByFile{
			{
				By: compute.By{
					StatementPercentage: 80.5,
					BlockPercentage:     75.0,
					StatementThreshold:  70.0,
					BlockThreshold:      60.0,
					Failed:              false,
				},
				File: "test.go",
			},
		},
		ByPackage: []compute.ByPackage{
			{
				By: compute.By{
					StatementPercentage: 80.5,
					BlockPercentage:     75.0,
					StatementThreshold:  70.0,
					BlockThreshold:      60.0,
					Failed:              false,
				},
				Package: "github.com/test/package",
			},
		},
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{
				Coverage:   "161/200",
				Percentage: 80.5,
				Threshold:  70.0,
				Failed:     false,
			},
			Blocks: compute.TotalBlocks{
				Coverage:   "75/100",
				Percentage: 75.0,
				Threshold:  60.0,
				Failed:     false,
			},
		},
	}

	cfg := &config.Config{}

	// Test successful coverage report
	result := FormatMarkdown(results, cfg, true)

	// Verify basic structure
	if !strings.Contains(result, "## 🚦 Coverage Report") {
		t.Error("Missing coverage report header")
	}
	if !strings.Contains(result, "Coverage check passed") {
		t.Error("Missing success message")
	}
	if !strings.Contains(result, "### 📊 Total Coverage") {
		t.Error("Missing total coverage section")
	}
	if !strings.Contains(result, "go-covercheck") {
		t.Error("Missing tool attribution")
	}

	// Test with failed coverage
	results.ByTotal.Statements.Failed = true
	results.ByFile[0].By.Failed = true
	results.ByPackage[0].By.Failed = true

	result = FormatMarkdown(results, cfg, true)

	if !strings.Contains(result, "Coverage check failed") {
		t.Error("Missing failure message")
	}
	if !strings.Contains(result, "### 📋 Coverage Details") {
		t.Error("Missing coverage details section")
	}
	if !strings.Contains(result, "### 💡 Required Improvements") {
		t.Error("Missing improvements section")
	}
}

func TestNewPoster(t *testing.T) {
	tests := []struct {
		platform string
		wantErr  bool
	}{
		{"github", false},
		{"gitlab", false},
		{"gitea", false},
		{"unsupported", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			_, err := NewPoster(tt.platform, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPoster() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}