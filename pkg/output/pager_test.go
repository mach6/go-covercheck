package output

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func updatePager(t *testing.T, m pagerModel, msg tea.Msg) (pagerModel, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	next, ok := updated.(pagerModel)
	require.True(t, ok, "Update should return pagerModel")
	return next, cmd
}

func TestPagerModel_QuitKey(t *testing.T) {
	m := newPagerModel("hello\nworld\n", "title")

	m, _ = updatePager(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd, "q key should return a quit command")

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "esc should return a quit command")
}

func TestPagerModel_WindowSizeInitializesViewport(t *testing.T) {
	m := newPagerModel("line1\nline2\n", "title")
	require.False(t, m.ready)

	m, _ = updatePager(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	require.True(t, m.ready)
	require.Equal(t, 100, m.viewport.Width)
}

func TestPagerModel_ViewContainsHeaderAndFooter(t *testing.T) {
	m := newPagerModel("some content\n", "go-covercheck — uncovered")
	m, _ = updatePager(t, m, tea.WindowSizeMsg{Width: 80, Height: 10})

	view := m.View()
	require.Contains(t, view, "uncovered")
	require.Contains(t, view, "q quit")
}

func TestPagerModel_TabsExpanded(t *testing.T) {
	m := newPagerModel("a\tb", "t")
	require.NotContains(t, m.content, "\t")
}

func TestPagerModel_InitReturnsNil(t *testing.T) {
	m := newPagerModel("x", "t")
	require.Nil(t, m.Init())
}

func TestPagerModel_ViewBeforeReady(t *testing.T) {
	m := newPagerModel("x", "t")
	require.Contains(t, m.View(), "initializing")
}

func TestPagerModel_CtrlCQuits(t *testing.T) {
	m := newPagerModel("x", "t")
	m, _ = updatePager(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)
}

func TestPagerModel_NavigationKeys(t *testing.T) {
	content := strings.Repeat("line\n", 100)
	m := newPagerModel(content, "t")
	m, _ = updatePager(t, m, tea.WindowSizeMsg{Width: 80, Height: 10})

	m, _ = updatePager(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	require.Positive(t, m.viewport.YOffset)

	m, _ = updatePager(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	require.Equal(t, 0, m.viewport.YOffset)

	m, _ = updatePager(t, m, tea.KeyMsg{Type: tea.KeyEnd})
	require.Positive(t, m.viewport.YOffset)

	m, _ = updatePager(t, m, tea.KeyMsg{Type: tea.KeyHome})
	require.Equal(t, 0, m.viewport.YOffset)
}

func TestPagerModel_WindowResize(t *testing.T) {
	m := newPagerModel("one\ntwo\n", "t")
	m, _ = updatePager(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = updatePager(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, 120, m.viewport.Width)
}

func TestPagerModel_WindowTooSmallClampsHeight(t *testing.T) {
	m := newPagerModel("x", "t")
	m, _ = updatePager(t, m, tea.WindowSizeMsg{Width: 20, Height: 1})
	require.True(t, m.ready)
	require.GreaterOrEqual(t, m.viewport.Height, 1)
}
