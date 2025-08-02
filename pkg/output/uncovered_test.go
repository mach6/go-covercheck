package output_test

import (
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestShowUncoveredLines_NoUncoveredLines(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.ShowUncovered = true
	cfg.NoColor = true
	color.NoColor = true

	profiles := []*cover.Profile{
		{
			FileName: "example.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 5, Count: 1, StartLine: 1, EndLine: 5},
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		err := output.ShowUncoveredLines(profiles, cfg)
		require.NoError(t, err)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "No uncovered lines found!")
}

func TestShowUncoveredLines_WithUncoveredLines(t *testing.T) {
	// Create a temporary test file
	testFile := "/tmp/test_uncovered.go"
	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello")
	if false {
		fmt.Println("Never reached")
	}
	fmt.Println("End")
}
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.ShowUncovered = true
	cfg.NoColor = true
	color.NoColor = true

	profiles := []*cover.Profile{
		{
			FileName: testFile,
			Blocks: []cover.ProfileBlock{
				{NumStmt: 1, Count: 1, StartLine: 6, EndLine: 6},   // Covered line
				{NumStmt: 1, Count: 0, StartLine: 8, EndLine: 8},   // Uncovered line
				{NumStmt: 1, Count: 1, StartLine: 10, EndLine: 10}, // Covered line
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		err := output.ShowUncoveredLines(profiles, cfg)
		require.NoError(t, err)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, testFile)
	require.Contains(t, stdout, "@@ Lines 8-8 (uncovered) @@")
	require.Contains(t, stdout, `fmt.Println("Never reached")`)
}

func TestShowUncoveredLines_SpecificFile(t *testing.T) {
	// Create a temporary test file
	testFile1 := "/tmp/test_uncovered1.go"
	testFile2 := "/tmp/test_uncovered2.go"
	testContent := `package main

func uncoveredFunc() {
	panic("This is uncovered")
}
`
	err := os.WriteFile(testFile1, []byte(testContent), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile1)

	err = os.WriteFile(testFile2, []byte(testContent), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile2)

	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.ShowUncovered = true
	cfg.UncoveredFile = "uncovered1.go"
	cfg.NoColor = true
	color.NoColor = true

	profiles := []*cover.Profile{
		{
			FileName: testFile1,
			Blocks: []cover.ProfileBlock{
				{NumStmt: 1, Count: 0, StartLine: 4, EndLine: 4}, // Uncovered line
			},
		},
		{
			FileName: testFile2,
			Blocks: []cover.ProfileBlock{
				{NumStmt: 1, Count: 0, StartLine: 4, EndLine: 4}, // Uncovered line
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		err := output.ShowUncoveredLines(profiles, cfg)
		require.NoError(t, err)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, testFile1)
	require.NotContains(t, stdout, testFile2) // Should only show the specific file
	require.Contains(t, stdout, "This is uncovered")
}

func TestShowUncoveredLines_FileNotFound(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.ShowUncovered = true
	cfg.UncoveredFile = "nonexistent.go"
	cfg.NoColor = true

	profiles := []*cover.Profile{
		{
			FileName: "other.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 1, Count: 0, StartLine: 1, EndLine: 1},
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		err := output.ShowUncoveredLines(profiles, cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found in coverage data")
	})

	require.Empty(t, stdout)
	require.Empty(t, stderr)
}

func TestShowUncoveredLines_WithColor(t *testing.T) {
	// Create a temporary test file
	testFile := "/tmp/test_uncovered_color.go"
	testContent := `package main

func uncoveredFunc() {
	return // This is uncovered
}
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.ShowUncovered = true
	cfg.NoColor = false
	color.NoColor = false

	profiles := []*cover.Profile{
		{
			FileName: testFile,
			Blocks: []cover.ProfileBlock{
				{NumStmt: 1, Count: 0, StartLine: 4, EndLine: 4}, // Uncovered line
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		err := output.ShowUncoveredLines(profiles, cfg)
		require.NoError(t, err)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, testFile)
	require.Contains(t, stdout, "return") // Should contain the uncovered line
	// Should contain color escape sequences if color is enabled
	if !strings.Contains(os.Getenv("TERM"), "dumb") {
		require.True(t, strings.Contains(stdout, "\x1b[") || strings.Contains(stdout, "\033["))
	}
}

func TestShowUncoveredLines_FilteringBehavior(t *testing.T) {
	// Create a test file with various line types that should be filtered
	testFile := "/tmp/test_filtering_behavior.go"
	testContent := `package main

import "fmt"

func testFunc() {
	// This comment should be filtered
	fmt.Println("actual code")
	
	/* Block comment start
	   Block comment middle
	   Block comment end */
	if true {
		fmt.Println("more code")
	} // Comment after closing brace
}`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.ShowUncovered = true
	cfg.NoColor = true
	color.NoColor = true

	// Mark all lines as uncovered to test filtering
	profiles := []*cover.Profile{
		{
			FileName: testFile,
			Blocks: []cover.ProfileBlock{
				{NumStmt: 1, Count: 0, StartLine: 1, EndLine: 15}, // All lines uncovered
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		err := output.ShowUncoveredLines(profiles, cfg)
		require.NoError(t, err)
	})

	require.Empty(t, stderr)
	
	// Should contain actual code lines
	require.Contains(t, stdout, `fmt.Println("actual code")`)
	require.Contains(t, stdout, `fmt.Println("more code")`)
	require.Contains(t, stdout, "package main")
	require.Contains(t, stdout, `import "fmt"`)
	require.Contains(t, stdout, "func testFunc() {")
	require.Contains(t, stdout, "if true {")
	
	// Should NOT contain filtered lines
	require.NotContains(t, stdout, "// This comment should be filtered")
	require.NotContains(t, stdout, "/* Block comment start")
	require.NotContains(t, stdout, "Block comment middle")
	require.NotContains(t, stdout, "Block comment end */")
	require.NotContains(t, stdout, "} // Comment after closing brace")
}

func TestShowUncoveredLines_DarkStyle(t *testing.T) {
	// Create a test file 
	testFile := "/tmp/test_dark_style.go"
	testContent := `package main

func testFunc() {
	return // uncovered line
}`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	profiles := []*cover.Profile{
		{
			FileName: testFile,
			Blocks: []cover.ProfileBlock{
				{NumStmt: 1, Count: 0, StartLine: 4, EndLine: 4}, // Uncovered line
			},
		},
	}

	// Test with dark style disabled
	cfg1 := new(config.Config)
	cfg1.ApplyDefaults()
	cfg1.ShowUncovered = true
	cfg1.NoColor = false
	cfg1.SyntaxStyle = "github"
	color.NoColor = false

	stdout1, stderr1 := test.RepipeStdOutAndErrForTest(func() {
		err := output.ShowUncoveredLines(profiles, cfg1)
		require.NoError(t, err)
	})

	// Test with dark style enabled
	cfg2 := new(config.Config)
	cfg2.ApplyDefaults()
	cfg2.ShowUncovered = true
	cfg2.NoColor = false
	cfg2.SyntaxStyle = "github-dark"
	color.NoColor = false

	stdout2, stderr2 := test.RepipeStdOutAndErrForTest(func() {
		err := output.ShowUncoveredLines(profiles, cfg2)
		require.NoError(t, err)
	})

	require.Empty(t, stderr1)
	require.Empty(t, stderr2)
	
	// Both should contain the uncovered line
	require.Contains(t, stdout1, "return")
	require.Contains(t, stdout2, "return")
	
	// Both should have color codes if terminal supports color
	if !strings.Contains(os.Getenv("TERM"), "dumb") {
		hasColor1 := strings.Contains(stdout1, "\x1b[") || strings.Contains(stdout1, "\033[")
		hasColor2 := strings.Contains(stdout2, "\x1b[") || strings.Contains(stdout2, "\033[")
		require.True(t, hasColor1)
		require.True(t, hasColor2)
	}
}