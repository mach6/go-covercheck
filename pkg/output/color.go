package output

import (
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/math"
	"github.com/muesli/termenv"
)

var (
	autoSyntaxStyleOnce sync.Once
	autoSyntaxStyle     string
)

// resolveSyntaxStyle picks a concrete chroma style name, honoring an explicit
// user selection and otherwise probing the terminal background via termenv
// (OSC 11). termenv.HasDarkBackground() returns true only for terminals it can
// positively identify as dark, so we use github-dark for those and github
// everywhere else — including terminals that can't answer the query, which
// are treated the same as "light" here since any ambiguity at startup would
// otherwise make the default unstable. The auto result is memoized for the
// life of the process so repeated highlight calls (e.g. the same run
// formatting many source lines) stay consistent and cheap.
func resolveSyntaxStyle(name string) string {
	if name != "" && name != config.SyntaxStyleAuto {
		return name
	}
	autoSyntaxStyleOnce.Do(func() {
		if termenv.HasDarkBackground() {
			autoSyntaxStyle = "github-dark"
			return
		}
		autoSyntaxStyle = "github"
	})
	return autoSyntaxStyle
}

func severityColor(actual, goal float64) func(a ...interface{}) string {
	switch {
	case goal <= 0 && actual <= 0:
		return color.New(color.Reset).SprintFunc()
	case goal <= 0:
		return color.New(color.Reset).SprintFunc()
	}

	pct := math.PercentFloat(actual, goal)
	switch {
	case pct <= 50: //nolint:mnd
		return color.New(color.FgRed).SprintFunc()
	case pct <= 99: //nolint:mnd
		return color.New(color.FgYellow).SprintFunc()
	default:
		return color.New(color.FgGreen).SprintFunc()
	}
}

func highlightJSONSyntax(content string, cfg *config.Config) string {
	return highlightSyntax(content, "json", cfg)
}

func highlightYAMLSyntax(content string, cfg *config.Config) string {
	return highlightSyntax(content, "yaml", cfg)
}

func highlightSyntax(content string, lexerName string, cfg *config.Config) string {
	return newSyntaxHighlighter(lexerName, cfg).highlight(content)
}

// syntaxHighlighter caches the resolved chroma lexer, formatter, and style so
// callers that highlight many lines (e.g. --inspect output) avoid repeating
// lookups and allocations on every line.
type syntaxHighlighter struct {
	lexer     chroma.Lexer
	formatter chroma.Formatter
	style     *chroma.Style
	enabled   bool
}

func newSyntaxHighlighter(lexerName string, cfg *config.Config) *syntaxHighlighter {
	if cfg.NoColor {
		return &syntaxHighlighter{}
	}
	lexer := lexers.Get(lexerName)
	if lexer == nil {
		return &syntaxHighlighter{}
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	style := styles.Get(resolveSyntaxStyle(cfg.SyntaxStyle))
	if style == nil {
		style = styles.Fallback
	}
	return &syntaxHighlighter{lexer: lexer, formatter: formatter, style: style, enabled: true}
}

func (h *syntaxHighlighter) highlight(content string) string {
	if !h.enabled || content == "" {
		return content
	}

	iterator, err := h.lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var result strings.Builder
	if err := h.formatter.Format(&result, h.style, iterator); err != nil {
		return content
	}

	return strings.TrimSuffix(result.String(), "\n")
}
