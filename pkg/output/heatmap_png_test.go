package output_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func writeAndDecode(t *testing.T, results compute.Results) image.Image {
	t.Helper()
	path := filepath.Join(t.TempDir(), "heat.png")
	require.NoError(t, output.WriteHeatmapPNG(path, results))
	data, err := os.ReadFile(path) //nolint:gosec // test temp path
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	return img
}

// count returns the number of pixels in img of color c.
func count(img image.Image, c color.RGBA) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r, g, bl, a := img.At(x, y).RGBA(); r>>8 == uint32(c.R) && g>>8 == uint32(c.G) &&
				bl>>8 == uint32(c.B) && a>>8 == uint32(c.A) {
				n++
			}
		}
	}
	return n
}

func TestWriteHeatmapPNG_SeverityColors(t *testing.T) {
	results := compute.Results{
		ByFile: []compute.ByFile{
			heatByFile("met.go", 100, 70, false),
			heatByFile("failed-but-high.go", 90, 70, true), // never green: expect yellow
			heatByFile("low.go", 10, 70, false),
			heatByFile("nogoal.go", 90, 0, false),
		},
	}
	results.ByTotal.Statements.Coverage = "1/2"
	results.ByTotal.Statements.Percentage = 50
	results.ByTotal.Statements.Threshold = 70
	img := writeAndDecode(t, results)

	cell := 168 * 40 * 3 / 4 // text covers part of each cell
	// files: met (green), failed (yellow), low (red), nogoal (grey) plus
	// the total cell (50% of 70 is above half the goal, so yellow),
	// and a swatch of each level in the legend.
	require.GreaterOrEqual(t, count(img, output.HeatColor(output.HeatMet)), cell)
	require.Less(t, count(img, output.HeatColor(output.HeatMet)), 2*cell, "only one green cell")
	require.GreaterOrEqual(t, count(img, output.HeatColor(output.HeatMid)), 2*cell)
	require.GreaterOrEqual(t, count(img, output.HeatColor(output.HeatLow)), cell)
	require.GreaterOrEqual(t, count(img, output.HeatColor(output.HeatNone)), cell)
}

func TestWriteHeatmapPNG_FailedNeverGreen(t *testing.T) {
	img := writeAndDecode(t, compute.Results{
		ByFile: []compute.ByFile{heatByFile("a.go", 100, 70, true)},
	})
	// only the legend swatch is green
	require.Less(t, count(img, output.HeatColor(output.HeatMet)), 168*40/2)
}

func TestWriteHeatmapPNG_Dimensions(t *testing.T) {
	files := func(n int) compute.Results {
		var r compute.Results
		for i := range n {
			r.ByFile = append(r.ByFile, heatByFile(fmt.Sprintf("f%d.go", i), 50, 70, false))
		}
		return r
	}
	one := writeAndDecode(t, files(1)).Bounds()
	five := writeAndDecode(t, files(5)).Bounds()
	six := writeAndDecode(t, files(6)).Bounds()
	require.Equal(t, output.PNGWidth, one.Dx())
	require.Equal(t, one.Dy(), five.Dy(), "five cells share a row")
	require.Equal(t, five.Dy()+output.PNGRowStep, six.Dy(), "the sixth wraps to a new row")
	require.Equal(t, output.PNGWidth, six.Dx())
}

func TestWriteHeatmapPNG_LongAndUnicodeNames(t *testing.T) {
	img := writeAndDecode(t, compute.Results{ByFile: []compute.ByFile{
		heatByFile("a/very/long/path/that/never/fits/in/a/cell/handler.go", 50, 70, false),
		heatByFile("\u65e5\u672c\u8a9e.go", 50, 70, false),
	}})
	require.Equal(t, output.PNGWidth, img.Bounds().Dx())
}

func TestWriteHeatmapPNG_Empty(t *testing.T) {
	img := writeAndDecode(t, compute.Results{})
	require.Equal(t, output.PNGWidth, img.Bounds().Dx())
	require.Positive(t, img.Bounds().Dy())
	// the legend swatches are the only colored pixels
	require.Less(t, count(img, output.HeatColor(output.HeatLow)), 168*40/2)
}

func TestWriteHeatmapPNG_UnwritablePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "heat.png")
	err := output.WriteHeatmapPNG(path, compute.Results{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "create heat map")
	require.NoFileExists(t, path)
}

func TestWriteHeatmapPNG_TooTall(t *testing.T) {
	var r compute.Results
	for i := range 4000 {
		r.ByFile = append(r.ByFile, heatByFile(fmt.Sprintf("f%d.go", i), 50, 70, false))
	}
	path := filepath.Join(t.TempDir(), "heat.png")
	err := output.WriteHeatmapPNG(path, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pixel limit")
	require.NoFileExists(t, path)
}

func TestFormatAndReport_HeatmapPNGAlongsideReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heat.png")
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatJSON
	cfg.NoColor = true
	cfg.HeatmapPNG = path

	profiles := []*cover.Profile{{
		FileName: "example/foo.go",
		Blocks:   []cover.ProfileBlock{{NumStmt: 10, Count: 1}},
	}}
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		output.FormatAndReport(results, cfg, failed)
	})
	require.Empty(t, stderr)
	require.Contains(t, stdout, `"example/foo.go"`)
	require.NotContains(t, stdout, "png")
	require.FileExists(t, path)
}

func TestValidate_HeatmapPNG(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	cfg.HeatmapPNG = "out.PNG"
	require.NoError(t, cfg.Validate())
	cfg.HeatmapPNG = "out.jpg"
	require.Error(t, cfg.Validate())
}
