package tui

import (
	"fmt"
	"strings"

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
	log      *logger.Logger
	all      []logger.Entry
	visible  []logger.Entry
	filter   logFilter
	cursor   int
	offset   int
	loading  bool
	err      string
	width    int
	height   int
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

func (m logsModel) visibleHeight() int {
	h := m.height - 8
	if h < 3 {
		h = 3
	}
	return h
}

func (m *logsModel) scrollToCursor() {
	vh := m.visibleHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+vh {
		m.offset = m.cursor - vh + 1
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
		m.offset = 0
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
		m.offset = 0
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.scrollToCursor()
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.scrollToCursor()
			}
		case "g":
			m.cursor = 0
			m.offset = 0
		case "G":
			if len(m.visible) > 0 {
				m.cursor = len(m.visible) - 1
				m.scrollToCursor()
			}
		case "f":
			m.filter = (m.filter + 1) % logFilter(len(filterLabels))
			m.visible = make([]logger.Entry, 0, len(m.all))
			m.applyFilter()
			m.cursor = 0
			m.offset = 0
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
	return m, nil
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
		title += "  " + styles.DimItem.Render("loading...")
	}
	b.WriteString(title + "\n\n")

	if m.err != "" {
		b.WriteString(styles.Error.Render("✗ "+m.err) + "\n\n")
	}

	if len(m.visible) == 0 && !m.loading {
		b.WriteString(styles.DimItem.Render("  No log entries.") + "\n")
	} else {
		b.WriteString(m.renderTable())
	}

	b.WriteString("\n")
	b.WriteString(styles.Help.Render(
		"[j/k] navigate  [f] filter  [r] refresh  [c] clear  [g/G] top/bottom",
	))
	return b.String()
}

func (m logsModel) renderTable() string {
	if len(m.visible) == 0 {
		return ""
	}

	available := m.width - 8
	if available < 50 {
		available = 50
	}
	timeW := 8
	levelW := 9
	sectionW := 10
	zoneW := 18
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

	var b strings.Builder
	b.WriteString("  " + header + "\n")
	b.WriteString("  " + styles.DimItem.Render(strings.Repeat("─", available)) + "\n")

	vh := m.visibleHeight()
	end := m.offset + vh
	if end > len(m.visible) {
		end = len(m.visible)
	}

	for i := m.offset; i < end; i++ {
		e := m.visible[i]
		isSelected := i == m.cursor

		ts := e.Time.Format("15:04:05")
		lvlStyle := levelStyle(e.Level)
		level := runesTrunc(string(e.Level), levelW-1)
		section := runesTrunc(e.Section, sectionW-1)
		zone := runesTrunc(e.Zone, zoneW-1)
		msg := runesTrunc(e.Message, msgW-1)

		if isSelected {
			sel := lipgloss.NewStyle().Foreground(styles.Orange).Bold(true)
			row := "▸ " +
				sel.Width(timeW).Render(ts) +
				sel.Width(levelW).Render(level) +
				sel.Width(sectionW).Render(section) +
				sel.Width(zoneW).Render(zone) +
				sel.Width(msgW).Render(msg)
			b.WriteString(row + "\n")
		} else {
			row := "  " +
				styles.DimItem.Width(timeW).Render(ts) +
				lvlStyle.Width(levelW).Render(level) +
				styles.NormalItem.Width(sectionW).Render(section) +
				styles.DimItem.Width(zoneW).Render(zone) +
				styles.NormalItem.Width(msgW).Render(msg)
			b.WriteString(row + "\n")
		}
	}

	if len(m.visible) > vh {
		b.WriteString(styles.DimItem.Render(
			fmt.Sprintf("  %d-%d of %d", m.offset+1, end, len(m.visible)),
		) + "\n")
	}
	return b.String()
}

func (m *logsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
