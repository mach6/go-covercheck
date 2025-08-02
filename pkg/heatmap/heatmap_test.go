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
					Statements:         "19/20",
				},
			},
			{
				File: "test2.go", 
				By: compute.By{
					StatementPercentage: 75.0,
					Statements:         "15/20",
				},
			},
		},
		ByPackage: []compute.ByPackage{
			{
				Package: "pkg/test",
				By: compute.By{
					StatementPercentage: 85.0,
					Statements:         "34/40",
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
	assert.Contains(t, output, "Coverage Legend (highlighting improvement opportunities):")
	assert.Contains(t, output, "FILES BY COVERAGE (highlighting improvement opportunities):")
	assert.Contains(t, output, "PACKAGES BY COVERAGE (highlighting improvement opportunities):")
	assert.Contains(t, output, "OVERALL COVERAGE SUMMARY:")
	assert.Contains(t, output, "pkg/test")
	assert.Contains(t, output, "85.0%")
	assert.Contains(t, output, "Statement Coverage")
	assert.Contains(t, output, "Block Coverage")
	
	// Verify both files and packages are displayed in grid format
	assert.Contains(t, output, "test1.go")
	assert.Contains(t, output, "test2.go")
	assert.Contains(t, output, "95.0%")
	assert.Contains(t, output, "75.0%")
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
		{"excellent coverage", 95.0, "▓▓ "},
		{"good coverage", 80.0, "▒▒ "},
		{"fair coverage", 60.0, "░░ "},
		{"poor coverage", 40.0, "▓▓ "},
		{"critical coverage", 20.0, "██ "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := heatmap.generateCoverageCell(tt.percentage)
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

func TestASCIIHeatmap_ExtractFilename(t *testing.T) {
	cfg := &config.Config{NoColor: true}
	var buf bytes.Buffer
	heatmap := NewASCIIHeatmap(&buf, cfg)

	tests := []struct {
		name     string
		fullPath string
		expected string
	}{
		{"simple filename", "test.go", "test.go"},
		{"path with slash", "pkg/config/test.go", "test.go"},
		{"long path", "very/long/path/to/file.go", "file.go"},
		{"empty path", "", ""},
		{"path ending with slash", "pkg/", "pkg/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := heatmap.extractFilename(tt.fullPath)
			assert.Equal(t, tt.expected, result)
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