package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mrbooshehri/cfctl/logger"
	"github.com/mrbooshehri/cfctl/styles"
)

type logFilter int

const (
	filterAll logFilter = iota
	filterInfo
	filterSuccess
	filterWarn
	filterError
)

var filterLabels = []string{"ALL", "INFO", "SUCCESS", "WARN", "ERROR"}

type logsModel struct {
	log     *logger.Logger
	all     []logger.Entry
	visible []logger.Entry
	filter  logFilter
	cursor  int
	vp      viewport.Model
	ready   bool
	loading bool
	err     string
	width   int
	height  int
}

type logsLoadedMsg struct{ entries []logger.Entry }
type logsErrMsg struct{ err error }
type logsClearedMsg struct{}

func newLogsModel(log *logger.Logger) logsModel {
	return logsModel{log: log}
}

func (m logsModel) Init() tea.Cmd { return m.load() }

func (m logsModel) load() tea.Cmd {
	log := m.log
	return func() tea.Msg {
		entries, err := log.Read()
		if err != nil {
			return logsErrMsg{err}
		}
		return logsLoadedMsg{entries}
	}
}

func (m *logsModel) applyFilter() {
	if m.filter == filterAll {
		m.visible = m.all
		return
	}
	level := map[logFilter]logger.Level{
		filterInfo:    logger.LevelInfo,
		filterSuccess: logger.LevelSuccess,
		filterWarn:    logger.LevelWarn,
		filterError:   logger.LevelError,
	}[m.filter]

	m.visible = m.visible[:0]
	for _, e := range m.all {
		if e.Level == level {
			m.visible = append(m.visible, e)
		}
	}
}

// vpHeight is the number of lines the viewport occupies.
// Overhead: border(2) + sectionTitle+border+blank(3) + col-header+sep(2) + indicator(1) + blank(1) + help(1) = 10 fixed + 1 spare = 11
func (m logsModel) vpHeight() int {
	h := m.height - 11
	if h < 3 {
		h = 3
	}
	return h
}

// rebuildViewport re-renders all rows into the viewport and scrolls to cursor.
func (m *logsModel) rebuildViewport() {
	m.vp.SetContent(m.renderRows())
	m.followCursor()
}

// followCursor adjusts the viewport's Y offset so the cursor row is visible.
func (m *logsModel) followCursor() {
	if m.cursor < m.vp.YOffset {
		m.vp.SetYOffset(m.cursor)
	} else if m.cursor >= m.vp.YOffset+m.vp.Height {
		m.vp.SetYOffset(m.cursor - m.vp.Height + 1)
	}
}

func (m logsModel) Update(msg tea.Msg) (logsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case logsLoadedMsg:
		m.loading = false
		m.all = msg.entries
		m.visible = make([]logger.Entry, 0, len(m.all))
		m.applyFilter()
		m.cursor = 0
		if m.ready {
			m.rebuildViewport()
			m.vp.GotoTop()
		}
		return m, nil

	case logsErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil

	case logsClearedMsg:
		m.loading = false
		m.all = nil
		m.visible = nil
		m.cursor = 0
		if m.ready {
			m.vp.SetContent("")
			m.vp.GotoTop()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.rebuildViewport()
			}
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.rebuildViewport()
			}
			return m, nil
		case "g":
			m.cursor = 0
			m.rebuildViewport()
			m.vp.GotoTop()
			return m, nil
		case "G":
			if len(m.visible) > 0 {
				m.cursor = len(m.visible) - 1
				m.rebuildViewport()
			}
			return m, nil
		case "f":
			m.filter = (m.filter + 1) % logFilter(len(filterLabels))
			m.visible = make([]logger.Entry, 0, len(m.all))
			m.applyFilter()
			m.cursor = 0
			if m.ready {
				m.rebuildViewport()
				m.vp.GotoTop()
			}
			return m, nil
		case "r":
			m.loading = true
			return m, m.load()
		case "c":
			log := m.log
			m.loading = true
			return m, func() tea.Msg {
				_ = log.Clear()
				return logsClearedMsg{}
			}
		}
	}

	// Forward everything else (pgup/pgdn/ctrl+u/ctrl+d/mouse) to the viewport.
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func levelStyle(level logger.Level) lipgloss.Style {
	switch level {
	case logger.LevelSuccess:
		return lipgloss.NewStyle().Foreground(styles.Green).Bold(true)
	case logger.LevelError:
		return lipgloss.NewStyle().Foreground(styles.Red).Bold(true)
	case logger.LevelWarn:
		return lipgloss.NewStyle().Foreground(styles.Yellow).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(styles.DimGray)
	}
}

func (m logsModel) View() string {
	var b strings.Builder

	// Title + filter badge
	filterBadge := lipgloss.NewStyle().
		Foreground(styles.BgDark).
		Background(styles.Orange).
		Bold(true).
		Padding(0, 1).
		Render(filterLabels[m.filter])
	title := styles.SectionTitle.Render("Activity Log") + "  " + filterBadge
	if m.loading {
		title += "  " + styles.DimItem.Render("loading…")
	}
	b.WriteString(title + "\n\n")

	if m.err != "" {
		b.WriteString(styles.Error.Render("✗ "+m.err) + "\n")
	}

	if len(m.visible) == 0 && !m.loading {
		b.WriteString(styles.DimItem.Render("  No log entries.") + "\n")
	} else {
		// Fixed column header
		b.WriteString(m.columnHeader())
		// Scrollable rows via viewport
		b.WriteString(m.vp.View() + "\n")
		// Scroll position indicator
		if m.vp.TotalLineCount() > m.vp.Height {
			pct := int(m.vp.ScrollPercent() * 100)
			info := fmt.Sprintf("  %d/%d  %d%%", m.cursor+1, len(m.visible), pct)
			b.WriteString(styles.DimItem.Render(info) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.Help.Render(
		"[j/k] row  [pgup/pgdn] page  [ctrl+u/d] half-page  [f] filter  [r] refresh  [c] clear",
	))
	return b.String()
}

func (m logsModel) columnHeader() string {
	available := m.width - 8
	if available < 50 {
		available = 50
	}
	timeW, levelW, sectionW, zoneW := 8, 9, 10, 18
	msgW := available - timeW - levelW - sectionW - zoneW
	if msgW < 15 {
		msgW = 15
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.TableHeader.Width(timeW).Render("TIME"),
		styles.TableHeader.Width(levelW).Render("LEVEL"),
		styles.TableHeader.Width(sectionW).Render("SECTION"),
		styles.TableHeader.Width(zoneW).Render("ZONE"),
		styles.TableHeader.Width(msgW).Render("MESSAGE"),
	)
	return "  " + header + "\n" +
		"  " + styles.DimItem.Render(strings.Repeat("─", available)) + "\n"
}

// renderRows returns all visible rows as a single string for the viewport.
func (m logsModel) renderRows() string {
	if len(m.visible) == 0 {
		return ""
	}

	available := m.width - 8
	if available < 50 {
		available = 50
	}
	timeW, levelW, sectionW, zoneW := 8, 9, 10, 18
	msgW := available - timeW - levelW - sectionW - zoneW
	if msgW < 15 {
		msgW = 15
	}

	var b strings.Builder
	for i, e := range m.visible {
		ts := e.Time.Format("15:04:05")
		lvlStyle := levelStyle(e.Level)
		level := runesTrunc(string(e.Level), levelW-1)
		section := runesTrunc(e.Section, sectionW-1)
		zone := runesTrunc(e.Zone, zoneW-1)
		msg := runesTrunc(e.Message, msgW-1)

		if i == m.cursor {
			sel := lipgloss.NewStyle().Foreground(styles.Orange).Bold(true)
			b.WriteString("▸ " +
				sel.Width(timeW).Render(ts) +
				sel.Width(levelW).Render(level) +
				sel.Width(sectionW).Render(section) +
				sel.Width(zoneW).Render(zone) +
				sel.Width(msgW).Render(msg) + "\n")
		} else {
			b.WriteString("  " +
				styles.DimItem.Width(timeW).Render(ts) +
				lvlStyle.Width(levelW).Render(level) +
				styles.NormalItem.Width(sectionW).Render(section) +
				styles.DimItem.Width(zoneW).Render(zone) +
				styles.NormalItem.Width(msgW).Render(msg) + "\n")
		}
	}
	// Trim the trailing \n so strings.Split gives exactly N elements
	// (no empty trailing element), keeping TotalLineCount == len(visible).
	return strings.TrimRight(b.String(), "\n")
}

func (m *logsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.vp.Width = w - 2
	m.vp.Height = m.vpHeight()
	m.ready = true
	m.rebuildViewport()
}
