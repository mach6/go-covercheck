package heatmap

import (
	"fmt"
	"io"
	"os"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// Generator represents a heat map generator.
type Generator interface {
	Generate(results compute.Results) error
}

// NewGenerator creates a new heat map generator based on the format.
func NewGenerator(format string, cfg *config.Config) (Generator, io.WriteCloser, error) {
	switch format {
	case config.FormatHeatmapASCII:
		return NewASCIIHeatmap(os.Stdout, cfg), nopWriteCloser{os.Stdout}, nil
	case config.FormatHeatmapPNG:
		output := cfg.HeatmapOutput
		if output == "" {
			output = "coverage-heatmap.png"
		}
		
		file, err := os.Create(output)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create PNG file %s: %w", output, err)
		}
		
		return NewPNGHeatmap(file, cfg), file, nil
	default:
		return nil, nil, fmt.Errorf("unsupported heatmap format: %s", format)
	}
}

// nopWriteCloser wraps an io.Writer to make it an io.WriteCloser with a no-op Close method.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}