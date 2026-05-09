package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	cf "github.com/cloudflare/cloudflare-go"
	"github.com/mrbooshehri/cfctl/api"
	"github.com/mrbooshehri/cfctl/styles"
)

type appState int

const (
	stateSetup appState = iota
	stateLoadingZones
	stateMain
)

type section int

const (
	sectionDNS section = iota
	sectionFirewall
	sectionSSL
)

var sectionNames = []string{"DNS", "Firewall", "SSL/TLS"}

type zonesLoadedMsg struct{ zones []cf.Zone }
type zonesErrMsg struct{ err error }

type AppModel struct {
	state         appState
	setup         setupModel
	client        *api.Client
	zones         []cf.Zone
	zoneIdx       int // active zone (loaded in content)
	sectionIdx    int // active section (loaded in content)
	sidebarCursor int // cursor position in unified sidebar list
	focusSide     bool
	dns           dnsModel
	fw            fwModel
	ssl           sslModel
	sectionInit   [3]bool
	err           string
	width         int
	height        int
}

func New(token string) (AppModel, error) {
	m := AppModel{focusSide: true}

	if token == "" {
		m.state = stateSetup
		m.setup = newSetupModel()
		return m, nil
	}

	client, err := api.New(token)
	if err != nil {
		return m, err
	}
	m.client = client
	m.state = stateLoadingZones
	return m, nil
}

func (m AppModel) Init() tea.Cmd {
	switch m.state {
	case stateSetup:
		return m.setup.Init()
	case stateLoadingZones:
		return m.loadZones()
	}
	return nil
}

func (m AppModel) loadZones() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		zones, err := client.ListZones(context.Background())
		if err != nil {
			return zonesErrMsg{err}
		}
		return zonesLoadedMsg{zones}
	}
}

// sidebarTotal is the total number of navigable items in the sidebar.
func (m AppModel) sidebarTotal() int {
	return len(m.zones) + len(sectionNames)
}

// applySidebarCursor syncs zoneIdx/sectionIdx from sidebarCursor and returns
// any needed init command.
func (m *AppModel) applySidebarCursor() tea.Cmd {
	n := len(m.zones)
	if m.sidebarCursor < n {
		if m.zoneIdx != m.sidebarCursor {
			m.zoneIdx = m.sidebarCursor
			m.sectionInit = [3]bool{}
			return m.initCurrentSection()
		}
	} else {
		newSection := m.sidebarCursor - n
		if m.sectionIdx != newSection {
			m.sectionIdx = newSection
			return m.initCurrentSection()
		}
	}
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeSections()
		if m.state == stateSetup {
			var cmd tea.Cmd
			m.setup, cmd = m.setup.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	case tokenValidMsg:
		m.client = msg.client
		m.state = stateLoadingZones
		return m, m.loadZones()

	case zonesLoadedMsg:
		m.zones = msg.zones
		if len(m.zones) > 0 {
			m.state = stateMain
			cmd := m.initCurrentSection()
			return m, cmd
		}
		m.err = "No zones found for this token"
		m.state = stateMain
		return m, nil

	case zonesErrMsg:
		m.err = msg.err.Error()
		m.state = stateMain
		return m, nil
	}

	switch m.state {
	case stateSetup:
		return m.updateSetup(msg)
	case stateMain:
		return m.updateMain(msg)
	}
	return m, nil
}

func (m AppModel) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.setup, cmd = m.setup.Update(msg)
	return m, cmd
}

func (m AppModel) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, isKey := msg.(tea.KeyMsg)

	if isKey {
		switch key.String() {
		case "q":
			if !m.contentHasForm() {
				return m, tea.Quit
			}
		// 1-3 jump directly to a section from anywhere
		case "1":
			m.sectionIdx = 0
			m.sidebarCursor = len(m.zones) + 0
			cmd := m.initCurrentSection()
			return m, cmd
		case "2":
			m.sectionIdx = 1
			m.sidebarCursor = len(m.zones) + 1
			cmd := m.initCurrentSection()
			return m, cmd
		case "3":
			m.sectionIdx = 2
			m.sidebarCursor = len(m.zones) + 2
			cmd := m.initCurrentSection()
			return m, cmd
		// h/l switch panels (vim-style)
		case "h":
			if !m.focusSide && !m.contentHasForm() {
				m.focusSide = true
				return m, nil
			}
		case "l", "enter":
			if m.focusSide {
				m.focusSide = false
				return m, nil
			}
		case "tab":
			if !m.contentHasForm() {
				m.focusSide = !m.focusSide
				return m, nil
			}
		}

		if m.focusSide {
			switch key.String() {
			case "j", "down":
				if m.sidebarCursor < m.sidebarTotal()-1 {
					m.sidebarCursor++
					cmd := m.applySidebarCursor()
					return m, cmd
				}
			case "k", "up":
				if m.sidebarCursor > 0 {
					m.sidebarCursor--
					cmd := m.applySidebarCursor()
					return m, cmd
				}
			case "g":
				m.sidebarCursor = 0
				cmd := m.applySidebarCursor()
				return m, cmd
			case "G":
				m.sidebarCursor = m.sidebarTotal() - 1
				cmd := m.applySidebarCursor()
				return m, cmd
			}
			return m, nil
		}
	}

	return m.updateSection(msg)
}

// contentHasForm reports whether the active section has an open input form,
// so we can suppress panel-switching and quit shortcuts.
func (m AppModel) contentHasForm() bool {
	switch section(m.sectionIdx) {
	case sectionDNS:
		return m.dns.showForm
	case sectionFirewall:
		return m.fw.showForm
	}
	return false
}

func (m AppModel) currentZoneID() string {
	if len(m.zones) == 0 || m.zoneIdx >= len(m.zones) {
		return ""
	}
	return m.zones[m.zoneIdx].ID
}

func (m *AppModel) initCurrentSection() tea.Cmd {
	zoneID := m.currentZoneID()
	if zoneID == "" || m.sectionInit[m.sectionIdx] {
		return nil
	}
	m.sectionInit[m.sectionIdx] = true

	switch section(m.sectionIdx) {
	case sectionDNS:
		m.dns = newDNSModel(m.client, zoneID)
		m.dns.SetSize(m.contentWidth(), m.contentHeight())
		return m.dns.Init()
	case sectionFirewall:
		m.fw = newFWModel(m.client, zoneID)
		m.fw.SetSize(m.contentWidth(), m.contentHeight())
		return m.fw.Init()
	case sectionSSL:
		m.ssl = newSSLModel(m.client, zoneID)
		return m.ssl.Init()
	}
	return nil
}

func (m AppModel) updateSection(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch section(m.sectionIdx) {
	case sectionDNS:
		m.dns, cmd = m.dns.Update(msg)
	case sectionFirewall:
		m.fw, cmd = m.fw.Update(msg)
	case sectionSSL:
		var s sslModel
		s, cmd = m.ssl.Update(msg)
		m.ssl = s
	}
	return m, cmd
}

func (m *AppModel) resizeSections() {
	cw := m.contentWidth()
	ch := m.contentHeight()
	m.dns.SetSize(cw, ch)
	m.fw.SetSize(cw, ch)
}

func (m AppModel) sidebarWidth() int {
	w := m.width / 5
	if w < 18 {
		w = 18
	}
	if w > 30 {
		w = 30
	}
	return w
}

func (m AppModel) contentWidth() int  { return m.width - m.sidebarWidth() - 4 }
func (m AppModel) contentHeight() int { return m.height - 4 }

func (m AppModel) View() string {
	switch m.state {
	case stateSetup:
		return m.setup.View()
	case stateLoadingZones:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			styles.Title.Render("cfctl")+"\n\n"+styles.DimItem.Render("Loading zones..."),
		)
	case stateMain:
		return m.mainView()
	}
	return ""
}

func (m AppModel) mainView() string {
	sw := m.sidebarWidth()
	cw := m.contentWidth()
	ch := m.contentHeight()

	sideStyle := styles.SidebarStyle.Width(sw).Height(ch)
	contentStyle := styles.ContentStyle.Width(cw).Height(ch)
	if m.focusSide {
		sideStyle = styles.ActiveBorder.Width(sw).Height(ch)
	} else {
		contentStyle = styles.ActiveBorder.Width(cw).Height(ch)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		sideStyle.Render(m.sidebarView()),
		contentStyle.Render(m.contentView()),
	)

	return lipgloss.JoinVertical(lipgloss.Left, m.headerView(), body)
}

func (m AppModel) headerView() string {
	zoneName := "—"
	if len(m.zones) > 0 && m.zoneIdx < len(m.zones) {
		zoneName = m.zones[m.zoneIdx].Name
	}
	left := styles.Header.Render(" cfctl ") + "  " +
		styles.DimItem.Render("zone:") + " " +
		styles.NormalItem.Render(zoneName)

	hint := "[h/l] panels  [j/k] navigate  [1-3] sections  [q] quit"
	right := styles.Help.Render(hint)
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}
	return left + strings.Repeat(" ", gap) + right
}

var (
	cursorStyle = lipgloss.NewStyle().Foreground(styles.Orange).Bold(true)
	activeStyle = lipgloss.NewStyle().Foreground(styles.Orange)
)

func (m AppModel) sidebarView() string {
	var b strings.Builder
	sw := m.sidebarWidth()

	renderItem := func(cursorIdx int, label, arrow string) {
		isCursor := m.focusSide && m.sidebarCursor == cursorIdx
		isActive := (cursorIdx < len(m.zones) && m.zoneIdx == cursorIdx) ||
			(cursorIdx >= len(m.zones) && m.sectionIdx == cursorIdx-len(m.zones))

		var line string
		switch {
		case isCursor:
			line = cursorStyle.Render(arrow + label)
		case isActive:
			line = activeStyle.Render(arrow + label)
		default:
			line = styles.NormalItem.Render("  " + label)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString(styles.DimItem.Render("ZONES") + "\n")
	for i, z := range m.zones {
		name := z.Name
		if len(name) > sw-3 {
			name = name[:sw-6] + "..."
		}
		renderItem(i, name, "▸ ")
	}

	b.WriteString("\n" + styles.DimItem.Render("SECTIONS") + "\n")
	for i, name := range sectionNames {
		label := fmt.Sprintf("[%d] %s", i+1, name)
		renderItem(len(m.zones)+i, label, "▸ ")
	}

	if m.focusSide {
		b.WriteString("\n" + styles.Help.Render("j/k navigate  l/enter select"))
	} else {
		b.WriteString("\n" + styles.Help.Render("h → sidebar"))
	}

	return b.String()
}

func (m AppModel) contentView() string {
	if m.err != "" {
		return styles.Error.Render("✗ " + m.err)
	}
	if len(m.zones) == 0 {
		return styles.DimItem.Render("No zones found.")
	}
	switch section(m.sectionIdx) {
	case sectionDNS:
		return m.dns.View()
	case sectionFirewall:
		return m.fw.View()
	case sectionSSL:
		return m.ssl.View()
	}
	return ""
}
