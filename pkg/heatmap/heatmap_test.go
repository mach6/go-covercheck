package heatmap

import (
	"bytes"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestASCIIHeatmap_Generate(t *testing.T) {
	cfg := &config.Config{NoColor: true}
	var buf bytes.Buffer
	heatmap := NewASCIIHeatmap(&buf, cfg)

	// Create test data
	results := compute.Results{
		ByFile: []compute.ByFile{
			{
				File: "test1.go",
				By: compute.By{
					StatementPercentage: 95.0,
					Statements:          "19/20",
				},
			},
			{
				File: "test2.go",
				By: compute.By{
					StatementPercentage: 75.0,
					Statements:          "15/20",
				},
			},
		},
		ByPackage: []compute.ByPackage{
			{
				Package: "pkg/test",
				By: compute.By{
					StatementPercentage: 85.0,
					Statements:          "34/40",
				},
			},
		},
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{
				Coverage:   "34/40",
				Percentage: 85.0,
			},
			Blocks: compute.TotalBlocks{
				Coverage:   "30/35",
				Percentage: 85.7,
			},
		},
	}

	err := heatmap.Generate(results)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "COVERAGE GRID HEAT MAP")
	assert.Contains(t, output, "Coverage Legend:")
	assert.Contains(t, output, "By Files:")
	assert.Contains(t, output, "By Packages:")
	assert.Contains(t, output, "By Total:")
	assert.Contains(t, output, "g/test") // Package name truncated to last 6 chars
	assert.Contains(t, output, "85.0%")
	assert.Contains(t, output, "Statement Coverage")
	assert.Contains(t, output, "Block Coverage")

	// Verify both files and packages are displayed in grid format with new layout
	assert.Contains(t, output, "test2., test1.") // File names truncated to 6 chars
	assert.Contains(t, output, "g/test")         // Package name truncated to last 6 chars
	assert.Contains(t, output, "95")             // Percentage without % in cell
	assert.Contains(t, output, "75")             // Percentage without % in cell
}

func TestASCIIHeatmap_GenerateCoverageCell(t *testing.T) {
	cfg := &config.Config{NoColor: true}
	var buf bytes.Buffer
	heatmap := NewASCIIHeatmap(&buf, cfg)

	tests := []struct {
		name       string
		percentage float64
		expected   string
	}{
		{"excellent coverage", 95.0, "▓▓▓▓▓▓▓\n▓▓▓95▓▓\n▓▓▓▓▓▓▓"},
		{"good coverage", 80.0, "▒▒▒▒▒▒▒\n▒▒▒80▒▒\n▒▒▒▒▒▒▒"},
		{"fair coverage", 60.0, "░░░░░░░\n░░░60░░\n░░░░░░░"},
		{"poor coverage", 40.0, "▓▓▓▓▓▓▓\n▓▓▓40▓▓\n▓▓▓▓▓▓▓"},
		{"critical coverage", 20.0, "███████\n███20██\n███████"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := heatmap.generateCoverageCell7x3(tt.percentage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewGenerator(t *testing.T) {
	cfg := &config.Config{}

	t.Run("ASCII format", func(t *testing.T) {
		generator, writer, err := NewGenerator(config.FormatHeatmapASCII, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, generator)
		assert.NotNil(t, writer)
		assert.IsType(t, &ASCIIHeatmap{}, generator)
		writer.Close()
	})

	t.Run("Unsupported format", func(t *testing.T) {
		generator, writer, err := NewGenerator("unsupported", cfg)
		assert.Error(t, err)
		assert.Nil(t, generator)
		assert.Nil(t, writer)
		assert.Contains(t, err.Error(), "unsupported heatmap format")
	})
}

func TestASCIIHeatmap_TruncateFilename(t *testing.T) {
	cfg := &config.Config{NoColor: true}
	var buf bytes.Buffer
	heatmap := NewASCIIHeatmap(&buf, cfg)

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"short filename", "test.go", "test.g"}, // Still truncated to 6 chars
		{"exact 6 chars", "test12", "test12"},
		{"long filename", "verylongfilename.go", "verylo"},
		{"7 chars", "test123", "test12"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := heatmap.truncateFilename(tt.filename)
			assert.Equal(t, tt.expected, result)
			assert.True(t, len(result) <= 6)
		})
	}
}

func TestASCIIHeatmap_TruncatePackageName(t *testing.T) {
	cfg := &config.Config{NoColor: true}
	var buf bytes.Buffer
	heatmap := NewASCIIHeatmap(&buf, cfg)

	tests := []struct {
		name        string
		packageName string
		expected    string
	}{
		{"short package", "test", "test"},
		{"exact 6 chars", "test12", "test12"},
		{"long package", "github.com/mach6/go-covercheck/pkg/math", "g/math"},
		{"7 chars", "test123", "est123"}, // Last 6 characters
		{"very long", "very/long/package/name", "e/name"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := heatmap.truncatePackageName(tt.packageName)
			assert.Equal(t, tt.expected, result)
			assert.True(t, len(result) <= 6)
		})
	}
}

func TestASCIIHeatmap_AbbreviateText(t *testing.T) {
	cfg := &config.Config{NoColor: true}
	var buf bytes.Buffer
	heatmap := NewASCIIHeatmap(&buf, cfg)

	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{"short text", "test.go", 20, "test.go"},
		{"long text", "very_long_filename.go", 10, "very_long.."},
		{"exact length", "exact.go", 8, "exact.go"},
		{"very short limit", "test.go", 3, "tes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := heatmap.abbreviateText(tt.text, tt.maxLen)
			assert.True(t, len(result) <= tt.maxLen)
			if len(tt.text) > tt.maxLen && tt.maxLen > 3 {
				assert.Contains(t, result, "..")
			} else if len(tt.text) <= tt.maxLen {
				assert.Equal(t, tt.text, result)
			}
		})
	}
}
