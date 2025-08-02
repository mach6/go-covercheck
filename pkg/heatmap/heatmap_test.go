package heatmap

import (
	"bytes"
	"strings"
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
	assert.Contains(t, output, "COVERAGE HEAT MAP")
	assert.Contains(t, output, "Statement Coverage Legend:")
	assert.Contains(t, output, "PACKAGES BY STATEMENT COVERAGE:")
	assert.Contains(t, output, "OVERALL COVERAGE:")
	assert.Contains(t, output, "pkg/test")
	assert.Contains(t, output, "85.0%")
	assert.Contains(t, output, "Statement Coverage")
	assert.Contains(t, output, "Block Coverage")
	
	// Verify files are NOT displayed (packages only)
	assert.NotContains(t, output, "FILES BY COVERAGE:")
	assert.NotContains(t, output, "test1.go")
	assert.NotContains(t, output, "test2.go")
}

func TestASCIIHeatmap_GenerateCoverageBar(t *testing.T) {
	cfg := &config.Config{NoColor: true}
	var buf bytes.Buffer
	heatmap := NewASCIIHeatmap(&buf, cfg)

	tests := []struct {
		name       string
		percentage float64
		expected   string
	}{
		{"100 percent", 100.0, "[==========]"},
		{"90 percent", 90.0, "[========= ]"},
		{"50 percent", 50.0, "[=====     ]"},
		{"10 percent", 10.0, "[=         ]"},
		{"0 percent", 0.0, "[          ]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := heatmap.generateCoverageBar(tt.percentage)
			assert.Contains(t, result, tt.expected)
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

func TestASCIIHeatmap_TruncateFileName(t *testing.T) {
	cfg := &config.Config{NoColor: true}
	var buf bytes.Buffer
	heatmap := NewASCIIHeatmap(&buf, cfg)

	tests := []struct {
		name     string
		filename string
		maxLen   int
		expected string
	}{
		{"short filename", "test.go", 20, "test.go"},
		{"long filename", "very/long/path/to/file.go", 15, "very/long/pa..."},
		{"exact length", "exact.go", 8, "exact.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := heatmap.truncateFileName(tt.filename, tt.maxLen)
			assert.True(t, len(strings.TrimSpace(result)) <= tt.maxLen)
			if len(tt.filename) > tt.maxLen {
				assert.Contains(t, result, "...")
			}
		})
	}
}