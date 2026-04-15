package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const pagerHelp = "j/k scroll  space pgdn  b pgup  g/G top/bot  q quit"

var (
	pagerHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("6")).
				Bold(true).
				Padding(0, 1)
	pagerFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Padding(0, 1)
)

type pagerModel struct {
	viewport viewport.Model
	content  string
	title    string
	ready    bool
}

func newPagerModel(content, title string) pagerModel {
	return pagerModel{
		content: strings.ReplaceAll(content, "\t", "    "),
		title:   title,
	}
}

func (m pagerModel) Init() tea.Cmd { return nil }

//nolint:cyclop // linear dispatch over Bubble Tea message types; splitting hurts readability
func (m pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Quit keys always work, even before the first WindowSizeMsg lands.
		if s := msg.String(); s == "q" || s == "esc" || s == "ctrl+c" {
			return m, tea.Quit
		}
		// Other key handlers touch m.viewport; bail out until it's initialized.
		if !m.ready {
			return m, nil
		}
		switch msg.String() {
		case "g", "home":
			m.viewport.GotoTop()
			return m, nil
		case "G", "end":
			m.viewport.GotoBottom()
			return m, nil
		}
	case tea.WindowSizeMsg:
		headerH := lipgloss.Height(m.headerView())
		footerH := lipgloss.Height(m.footerView())
		vpHeight := msg.Height - headerH - footerH
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m pagerModel) View() string {
	if !m.ready {
		return "initializing…"
	}
	return m.headerView() + "\n" + m.viewport.View() + "\n" + m.footerView()
}

func (m pagerModel) headerView() string {
	return pagerHeaderStyle.Render(m.title)
}

func (m pagerModel) footerView() string {
	const pctFactor = 100
	pct := fmt.Sprintf("%3.0f%%", m.viewport.ScrollPercent()*pctFactor)
	return pagerFooterStyle.Render(pct + "  " + pagerHelp)
}

func runBubblePager(output string) error {
	m := newPagerModel(output, "go-covercheck — uncovered")
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
