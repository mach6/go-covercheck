package output

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
	"golang.org/x/tools/cover"
)

// UncoveredLine represents a line that is not covered by tests
type UncoveredLine struct {
	LineNumber int
	Content    string
	IsCovered  bool
}

// UncoveredBlock represents a block of uncovered lines with context
type UncoveredBlock struct {
	StartLine    int
	EndLine      int
	Lines        []UncoveredLine
	ContextLines []UncoveredLine // Context lines around the uncovered block
}

// FileUncoveredInfo contains uncovered information for a file
type FileUncoveredInfo struct {
	FileName string
	Blocks   []UncoveredBlock
}

// ShowUncoveredLines displays uncovered lines for the specified files
func ShowUncoveredLines(profiles []*cover.Profile, cfg *config.Config) error {
	var filesToShow []*cover.Profile

	// Filter profiles to show
	if cfg.UncoveredFile != "" {
		for _, profile := range profiles {
			if strings.Contains(profile.FileName, cfg.UncoveredFile) {
				filesToShow = append(filesToShow, profile)
			}
		}
		if len(filesToShow) == 0 {
			return fmt.Errorf("file %q not found in coverage data", cfg.UncoveredFile)
		}
	} else {
		filesToShow = profiles
	}

	uncoveredInfos := make([]FileUncoveredInfo, 0)
	for _, profile := range filesToShow {
		// Check if this profile has any uncovered blocks
		hasUncovered := false
		for _, block := range profile.Blocks {
			if block.Count == 0 {
				hasUncovered = true
				break
			}
		}
		
		if !hasUncovered {
			continue
		}
		
		info, err := analyzeFileUncovered(profile, cfg)
		if err != nil {
			continue // Skip files that can't be read
		}
		if len(info.Blocks) > 0 {
			uncoveredInfos = append(uncoveredInfos, info)
		}
	}

	if len(uncoveredInfos) == 0 {
		fmt.Println("No uncovered lines found!")
		return nil
	}

	return displayUncoveredLines(uncoveredInfos, cfg)
}

// analyzeFileUncovered analyzes a single file to find uncovered lines
func analyzeFileUncovered(profile *cover.Profile, cfg *config.Config) (FileUncoveredInfo, error) {
	info := FileUncoveredInfo{
		FileName: profile.FileName,
		Blocks:   make([]UncoveredBlock, 0),
	}

	// Read the source file
	sourceLines, err := readSourceFile(profile.FileName)
	if err != nil {
		return info, err
	}

	// Create a map to track covered lines
	coveredLines := make(map[int]bool)
	for _, block := range profile.Blocks {
		if block.Count > 0 {
			for line := block.StartLine; line <= block.EndLine; line++ {
				coveredLines[line] = true
			}
		}
	}

	// Find uncovered blocks
	var currentBlock *UncoveredBlock
	for _, block := range profile.Blocks {
		if block.Count == 0 {
			// This block is uncovered
			for line := block.StartLine; line <= block.EndLine; line++ {
				if line <= len(sourceLines) {
					uncoveredLine := UncoveredLine{
						LineNumber: line,
						Content:    sourceLines[line-1], // Convert to 0-based index
						IsCovered:  false,
					}

					// Group consecutive uncovered lines into blocks
					if currentBlock == nil || line != currentBlock.EndLine+1 {
						if currentBlock != nil {
							// Add context lines to the completed block
							addContextLines(currentBlock, sourceLines, coveredLines, cfg.UncoveredContext)
							info.Blocks = append(info.Blocks, *currentBlock)
						}
						currentBlock = &UncoveredBlock{
							StartLine: line,
							EndLine:   line,
							Lines:     []UncoveredLine{uncoveredLine},
						}
					} else {
						currentBlock.EndLine = line
						currentBlock.Lines = append(currentBlock.Lines, uncoveredLine)
					}
				}
			}
		}
	}

	if currentBlock != nil {
		// Add context lines to the final block
		addContextLines(currentBlock, sourceLines, coveredLines, cfg.UncoveredContext)
		info.Blocks = append(info.Blocks, *currentBlock)
	}

	return info, nil
}

// addContextLines adds context lines around an uncovered block
func addContextLines(block *UncoveredBlock, sourceLines []string, coveredLines map[int]bool, contextSize int) {
	if contextSize <= 0 {
		return
	}

	// Add lines before the uncovered block
	startContext := block.StartLine - contextSize
	if startContext < 1 {
		startContext = 1
	}

	// Add lines after the uncovered block  
	endContext := block.EndLine + contextSize
	if endContext > len(sourceLines) {
		endContext = len(sourceLines)
	}

	// Collect context lines
	for i := startContext; i < block.StartLine; i++ {
		contextLine := UncoveredLine{
			LineNumber: i,
			Content:    sourceLines[i-1],
			IsCovered:  coveredLines[i],
		}
		block.ContextLines = append(block.ContextLines, contextLine)
	}

	// Add the uncovered lines themselves to context
	block.ContextLines = append(block.ContextLines, block.Lines...)

	// Add trailing context
	for i := block.EndLine + 1; i <= endContext; i++ {
		contextLine := UncoveredLine{
			LineNumber: i,
			Content:    sourceLines[i-1],
			IsCovered:  coveredLines[i],
		}
		block.ContextLines = append(block.ContextLines, contextLine)
	}
}

// readSourceFile reads the content of a source file
func readSourceFile(fileName string) ([]string, error) {
	// Try the filename as-is first
	file, err := os.Open(fileName)
	if err != nil {
		// If that fails, try to find the file relative to current directory
		// Coverage data often has full package paths
		if strings.Contains(fileName, "/") {
			// Extract just the relative part after the module name
			parts := strings.Split(fileName, "/")
			if len(parts) >= 2 {
				// Try various combinations of the path components
				for i := 0; i < len(parts); i++ {
					relativePath := strings.Join(parts[i:], "/")
					file, err = os.Open(relativePath)
					if err == nil {
						break
					}
				}
			}
		}
		if err != nil {
			return nil, err
		}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// displayUncoveredLines displays the uncovered lines with formatting
func displayUncoveredLines(infos []FileUncoveredInfo, cfg *config.Config) error {
	// Sort files by name for consistent output
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].FileName < infos[j].FileName
	})

	output := buildUncoveredOutput(infos, cfg)
	
	// Use pager if output is large and we're in a terminal
	if shouldUsePager(output, cfg) {
		return displayWithPager(output)
	}

	fmt.Print(output)
	return nil
}

// buildUncoveredOutput builds the formatted output string
func buildUncoveredOutput(infos []FileUncoveredInfo, cfg *config.Config) string {
	var sb strings.Builder

	for i, info := range infos {
		if i > 0 {
			sb.WriteString("\n")
		}

		// File header
		fileHeader := fmt.Sprintf("--- %s", info.FileName)
		if !cfg.NoColor {
			fileHeader = color.New(color.FgCyan, color.Bold).Sprint(fileHeader)
		}
		sb.WriteString(fileHeader + "\n")

		// Show uncovered blocks
		for _, block := range info.Blocks {
			if !cfg.NoColor {
				blockHeader := color.New(color.FgYellow).Sprintf("@@ Lines %d-%d (uncovered) @@", 
					block.StartLine, block.EndLine)
				sb.WriteString(blockHeader + "\n")
			} else {
				sb.WriteString(fmt.Sprintf("@@ Lines %d-%d (uncovered) @@\n", 
					block.StartLine, block.EndLine))
			}

			// Show the context lines if available, otherwise show just the uncovered lines
			linesToShow := block.Lines
			if len(block.ContextLines) > 0 {
				linesToShow = block.ContextLines
			}

			for _, line := range linesToShow {
				lineStr := formatSourceLine(line, cfg)
				sb.WriteString(lineStr + "\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// formatSourceLine formats a source line with proper coloring and indicators
func formatSourceLine(line UncoveredLine, cfg *config.Config) string {
	lineNumStr := fmt.Sprintf("%4d", line.LineNumber)
	
	if cfg.NoColor {
		if line.IsCovered {
			return fmt.Sprintf("  %s: %s", lineNumStr, line.Content)
		}
		return fmt.Sprintf("- %s: %s", lineNumStr, line.Content)
	}

	// With color
	if line.IsCovered {
		lineNumStr = color.New(color.FgGreen).Sprint(lineNumStr)
		content := highlightGoSyntax(line.Content)
		return fmt.Sprintf("  %s: %s", lineNumStr, content)
	}

	// Uncovered line
	lineNumStr = color.New(color.FgRed).Sprint(lineNumStr)
	content := highlightGoSyntax(line.Content)
	return fmt.Sprintf("- %s: %s", lineNumStr, content)
}

// highlightGoSyntax provides basic Go syntax highlighting
func highlightGoSyntax(content string) string {
	if content == "" {
		return content
	}

	// Basic Go keywords with word boundaries
	keywords := map[string]color.Attribute{
		"package":   color.FgMagenta,
		"import":    color.FgMagenta,
		"func":      color.FgBlue,
		"var":       color.FgBlue,
		"const":     color.FgBlue,
		"type":      color.FgBlue,
		"struct":    color.FgBlue,
		"interface": color.FgBlue,
		"if":        color.FgYellow,
		"else":      color.FgYellow,
		"for":       color.FgYellow,
		"switch":    color.FgYellow,
		"case":      color.FgYellow,
		"default":   color.FgYellow,
		"return":    color.FgYellow,
		"break":     color.FgYellow,
		"continue":  color.FgYellow,
		"go":        color.FgYellow,
		"defer":     color.FgYellow,
		"select":    color.FgYellow,
		"chan":      color.FgCyan,
		"map":       color.FgCyan,
		"range":     color.FgYellow,
	}

	result := content

	// Handle string literals first (to avoid highlighting keywords inside strings)
	if strings.Contains(result, `"`) {
		// Simple string highlighting - this is basic but functional
		result = strings.ReplaceAll(result, `"`, color.New(color.FgGreen).Sprint(`"`))
	}

	// Highlight comments
	if strings.Contains(result, "//") {
		parts := strings.Split(result, "//")
		if len(parts) > 1 {
			commentPart := "//" + strings.Join(parts[1:], "//")
			highlighted := color.New(color.FgGreen, color.Italic).Sprint(commentPart)
			result = parts[0] + highlighted
		}
	}

	// Highlight keywords
	for keyword, colorAttr := range keywords {
		// Use word boundaries to avoid partial matches
		patterns := []string{
			keyword + " ",
			keyword + "\t",
			keyword + "(",
			keyword + "{",
			keyword + "\n",
			" " + keyword + " ",
			" " + keyword + "\t",
			" " + keyword + "(",
			" " + keyword + "{",
			"\t" + keyword + " ",
			"\t" + keyword + "\t",
			"\t" + keyword + "(",
			"\t" + keyword + "{",
		}

		for _, pattern := range patterns {
			if strings.Contains(result, pattern) {
				highlighted := color.New(colorAttr, color.Bold).Sprint(keyword)
				replacement := strings.ReplaceAll(pattern, keyword, highlighted)
				result = strings.ReplaceAll(result, pattern, replacement)
			}
		}

		// Handle keyword at start of line
		if strings.HasPrefix(strings.TrimSpace(result), keyword+" ") ||
			strings.HasPrefix(strings.TrimSpace(result), keyword+"\t") ||
			strings.HasPrefix(strings.TrimSpace(result), keyword+"(") ||
			strings.HasPrefix(strings.TrimSpace(result), keyword+"{") {
			highlighted := color.New(colorAttr, color.Bold).Sprint(keyword)
			result = strings.Replace(result, keyword, highlighted, 1)
		}
	}

	return result
}

// shouldUsePager determines if we should use a pager for output
func shouldUsePager(output string, cfg *config.Config) bool {
	lines := strings.Count(output, "\n")
	return lines > 30 && isTerminal() // Use pager for output with more than 30 lines
}

// isTerminal checks if we're running in a terminal
func isTerminal() bool {
	if os.Getenv("TERM") == "" {
		return false
	}
	// Check if stdout is a terminal
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// displayWithPager displays output using a pager (less/more)
func displayWithPager(output string) error {
	// Try to use 'less' first, then 'more'
	pagers := []string{"less", "more"}
	
	for _, pager := range pagers {
		if _, err := exec.LookPath(pager); err == nil {
			cmd := exec.Command(pager)
			cmd.Stdin = strings.NewReader(output)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			if pager == "less" {
				// Add some useful less options
				cmd.Env = append(os.Environ(), "LESS=-R") // Enable color support
			}
			
			return cmd.Run()
		}
	}
	
	// If no pager is available, just print to stdout
	fmt.Print(output)
	return nil
}