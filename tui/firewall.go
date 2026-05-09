package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	cf "github.com/cloudflare/cloudflare-go"
	"github.com/mrbooshehri/cfctl/api"
	"github.com/mrbooshehri/cfctl/logger"
	"github.com/mrbooshehri/cfctl/styles"
)

type fwModel struct {
	client      *api.Client
	log         *logger.Logger
	zoneID      string
	zoneName    string
	rules       []cf.AccessRule
	cursor      int
	offset      int
	showForm    bool
	showConfirm bool
	confirmIdx  int
	ipInput     textinput.Model
	modeInput   textinput.Model
	noteInput   textinput.Model
	formFocus   int
	loading     bool
	err         string
	statusMsg   string
	width       int
	height      int
}

type fwLoadedMsg struct{ rules []cf.AccessRule }
type fwErrMsg struct{ err error }
type fwOKMsg struct{ msg string }

func newFWModel(client *api.Client, log *logger.Logger, zoneID, zoneName string) fwModel {
	ip := textinput.New()
	ip.Placeholder = "IP / CIDR / country code"
	ip.Width = 45

	mode := textinput.New()
	mode.Placeholder = "block / challenge / whitelist / js_challenge"
	mode.Width = 45

	note := textinput.New()
	note.Placeholder = "Optional note"
	note.Width = 45

	return fwModel{
		client:    client,
		log:       log,
		zoneID:    zoneID,
		zoneName:  zoneName,
		ipInput:   ip,
		modeInput: mode,
		noteInput: note,
	}
}

func (m fwModel) Init() tea.Cmd { return m.load() }

func (m fwModel) load() tea.Cmd {
	client, log, zoneID, zoneName := m.client, m.log, m.zoneID, m.zoneName
	return func() tea.Msg {
		rules, err := client.ListAccessRules(context.Background(), zoneID)
		if err != nil {
			if log != nil {
				log.Write(logger.LevelError, "Firewall", zoneName, "List rules: "+fwErrString(err))
			}
			return fwErrMsg{err}
		}
		if log != nil {
			log.Write(logger.LevelInfo, "Firewall", zoneName, fmt.Sprintf("Listed %d access rules", len(rules)))
		}
		return fwLoadedMsg{rules}
	}
}

func (m fwModel) visibleHeight() int {
	h := m.height - 11
	if h < 3 {
		h = 3
	}
	return h
}

func (m *fwModel) scrollToCursor() {
	vh := m.visibleHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+vh {
		m.offset = m.cursor - vh + 1
	}
}

func (m fwModel) Update(msg tea.Msg) (fwModel, tea.Cmd) {
	switch msg := msg.(type) {
	case fwLoadedMsg:
		m.loading = false
		m.rules = msg.rules
		m.cursor = 0
		m.offset = 0
		return m, nil

	case fwErrMsg:
		m.loading = false
		m.err = fwErrString(msg.err)
		return m, nil

	case fwOKMsg:
		m.statusMsg = msg.msg
		m.showForm = false
		return m, m.load()

	case tea.KeyMsg:
		if m.showForm {
			return m.updateForm(msg)
		}
		if m.showConfirm {
			return m.updateConfirm(msg)
		}
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.rules)-1 {
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
			if len(m.rules) > 0 {
				m.cursor = len(m.rules) - 1
				m.scrollToCursor()
			}
		case "n":
			m.ipInput.SetValue("")
			m.modeInput.SetValue("block")
			m.noteInput.SetValue("")
			m.formFocus = 0
			m.ipInput.Focus()
			m.modeInput.Blur()
			m.noteInput.Blur()
			m.showForm = true
			m.err = ""
			m.statusMsg = ""
		case "d":
			if len(m.rules) > 0 && m.cursor < len(m.rules) {
				m.showConfirm = true
				m.confirmIdx = m.cursor
				m.err = ""
				m.statusMsg = ""
			}
		case "r":
			m.loading = true
			return m, m.load()
		}
	}
	return m, nil
}

func (m fwModel) updateForm(msg tea.KeyMsg) (fwModel, tea.Cmd) {
	inputs := []*textinput.Model{&m.ipInput, &m.modeInput, &m.noteInput}
	switch msg.Type {
	case tea.KeyEsc:
		m.showForm = false
		return m, nil
	case tea.KeyTab, tea.KeyDown:
		inputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus + 1) % len(inputs)
		inputs[m.formFocus].Focus()
		return m, textinput.Blink
	case tea.KeyShiftTab, tea.KeyUp:
		inputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus - 1 + len(inputs)) % len(inputs)
		inputs[m.formFocus].Focus()
		return m, textinput.Blink
	case tea.KeyEnter:
		if m.formFocus < len(inputs)-1 {
			inputs[m.formFocus].Blur()
			m.formFocus++
			inputs[m.formFocus].Focus()
			return m, textinput.Blink
		}
		return m.submitForm()
	}
	var cmd tea.Cmd
	switch m.formFocus {
	case 0:
		m.ipInput, cmd = m.ipInput.Update(msg)
	case 1:
		m.modeInput, cmd = m.modeInput.Update(msg)
	case 2:
		m.noteInput, cmd = m.noteInput.Update(msg)
	}
	return m, cmd
}

func (m fwModel) submitForm() (fwModel, tea.Cmd) {
	value := strings.TrimSpace(m.ipInput.Value())
	mode := strings.TrimSpace(m.modeInput.Value())
	note := strings.TrimSpace(m.noteInput.Value())

	if value == "" {
		m.err = "IP/CIDR/country is required"
		return m, nil
	}
	if mode == "" {
		mode = "block"
	}

	target := "ip"
	if strings.Contains(value, "/") {
		target = "ip_range"
	} else if len(value) == 2 && !strings.Contains(value, ".") && !strings.Contains(value, ":") {
		target = "country"
	}

	rule := cf.AccessRule{
		Mode:  mode,
		Notes: note,
		Configuration: cf.AccessRuleConfiguration{Target: target, Value: value},
	}
	client, log, zoneID, zoneName := m.client, m.log, m.zoneID, m.zoneName

	return m, func() tea.Msg {
		if _, err := client.CreateAccessRule(context.Background(), zoneID, rule); err != nil {
			if log != nil {
				log.Write(logger.LevelError, "Firewall", zoneName, "Create rule: "+err.Error())
			}
			return fwErrMsg{err}
		}
		if log != nil {
			log.Write(logger.LevelSuccess, "Firewall", zoneName,
				fmt.Sprintf("Created %s rule for %s (%s)", mode, value, target))
		}
		return fwOKMsg{"Rule created"}
	}
}

func (m fwModel) View() string {
	if m.showForm {
		return m.formView()
	}
	if m.showConfirm {
		return m.confirmView()
	}

	var b strings.Builder
	title := styles.SectionTitle.Render("Firewall — IP Access Rules")
	if m.loading {
		b.WriteString(title + "  " + styles.DimItem.Render("loading...") + "\n\n")
	} else {
		b.WriteString(title + "\n\n")
	}

	if m.err != "" {
		b.WriteString(styles.Error.Render("✗ "+m.err) + "\n\n")
	}
	if m.statusMsg != "" {
		b.WriteString(styles.Success.Render("✓ "+m.statusMsg) + "\n\n")
	}

	b.WriteString(m.renderTable())
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("[n] new  [d] delete  [r] refresh  [j/k] navigate  [g/G] top/bottom"))

	return b.String()
}

func (m fwModel) renderTable() string {
	if len(m.rules) == 0 {
		return styles.DimItem.Render("  No access rules found.") + "\n"
	}

	available := m.width - 8
	if available < 40 {
		available = 40
	}
	modeW := 11
	targetW := 10
	valueW := 20
	noteW := available - modeW - targetW - valueW
	if noteW < 8 {
		noteW = 8
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.TableHeader.Width(modeW).Render("MODE"),
		styles.TableHeader.Width(valueW).Render("VALUE"),
		styles.TableHeader.Width(targetW).Render("TARGET"),
		styles.TableHeader.Width(noteW).Render("NOTES"),
	)

	var b strings.Builder
	b.WriteString("  " + header + "\n")
	b.WriteString("  " + styles.DimItem.Render(strings.Repeat("─", available)) + "\n")

	vh := m.visibleHeight()
	end := m.offset + vh
	if end > len(m.rules) {
		end = len(m.rules)
	}

	sel := lipgloss.NewStyle().Foreground(styles.Orange).Bold(true)
	normal := lipgloss.NewStyle().Foreground(styles.White)
	dim := lipgloss.NewStyle().Foreground(styles.DimGray)

	for i := m.offset; i < end; i++ {
		r := m.rules[i]
		isSelected := i == m.cursor

		value := runesTrunc(r.Configuration.Value, valueW-1)
		notes := runesTrunc(r.Notes, noteW-1)

		if isSelected {
			row := "▸ " + sel.Width(modeW).Render(r.Mode) +
				sel.Width(valueW).Render(value) +
				sel.Width(targetW).Render(r.Configuration.Target) +
				sel.Width(noteW).Render(notes)
			b.WriteString(row + "\n")
		} else {
			row := "  " + normal.Width(modeW).Render(r.Mode) +
				dim.Width(valueW).Render(value) +
				dim.Width(targetW).Render(r.Configuration.Target) +
				dim.Width(noteW).Render(notes)
			b.WriteString(row + "\n")
		}
	}

	if len(m.rules) > vh {
		b.WriteString(styles.DimItem.Render(fmt.Sprintf(
			"  %d-%d of %d", m.offset+1, end, len(m.rules))) + "\n")
	}

	return b.String()
}

func (m fwModel) formView() string {
	inputs := []textinput.Model{m.ipInput, m.modeInput, m.noteInput}
	var b strings.Builder
	b.WriteString(styles.SectionTitle.Render("New Access Rule") + "\n\n")
	for i, f := range inputs {
		if i == m.formFocus {
			b.WriteString(styles.SelectedItem.Render("> ") + f.View() + "\n")
		} else {
			b.WriteString("  " + f.View() + "\n")
		}
	}
	b.WriteString("\n")
	if m.err != "" {
		b.WriteString(styles.Error.Render("✗ "+m.err) + "\n\n")
	}
	b.WriteString(styles.Help.Render("[tab/↑↓] move  [enter] next/submit  [esc] cancel"))
	return b.String()
}

func (m fwModel) updateConfirm(msg tea.KeyMsg) (fwModel, tea.Cmd) {
	switch {
	case msg.String() == "y" || msg.String() == "Y":
		if m.confirmIdx >= len(m.rules) {
			m.showConfirm = false
			return m, nil
		}
		r := m.rules[m.confirmIdx]
		client, log, zoneID, zoneName := m.client, m.log, m.zoneID, m.zoneName
		m.showConfirm = false
		m.loading = true
		return m, func() tea.Msg {
			if err := client.DeleteAccessRule(context.Background(), zoneID, r.ID); err != nil {
				if log != nil {
					log.Write(logger.LevelError, "Firewall", zoneName, "Delete rule: "+err.Error())
				}
				return fwErrMsg{err}
			}
			if log != nil {
				log.Write(logger.LevelSuccess, "Firewall", zoneName,
					fmt.Sprintf("Deleted %s rule for %s (%s)", r.Mode, r.Configuration.Value, r.Configuration.Target))
			}
			return fwOKMsg{"Rule deleted"}
		}
	default:
		m.showConfirm = false
	}
	return m, nil
}

func (m fwModel) confirmView() string {
	if m.confirmIdx >= len(m.rules) {
		return ""
	}
	r := m.rules[m.confirmIdx]

	var b strings.Builder
	b.WriteString(styles.Error.Render("Delete Access Rule?") + "\n\n")
	b.WriteString(styles.DimItem.Render("  This action cannot be undone.") + "\n\n")
	b.WriteString("  " + styles.DimItem.Render("Mode:   ") + styles.NormalItem.Render(r.Mode) + "\n")
	b.WriteString("  " + styles.DimItem.Render("Target: ") + styles.NormalItem.Render(r.Configuration.Target) + "\n")
	b.WriteString("  " + styles.DimItem.Render("Value:  ") + styles.NormalItem.Render(r.Configuration.Value) + "\n")
	if r.Notes != "" {
		b.WriteString("  " + styles.DimItem.Render("Notes:  ") + styles.NormalItem.Render(r.Notes) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(styles.Error.Render("[y] confirm delete") + "  " + styles.Help.Render("[any other key] cancel"))
	return b.String()
}

// fwErrString converts a Cloudflare API error into a human-readable message,
// catching the common case where the token lacks Firewall Services permission.
func fwErrString(err error) string {
	s := err.Error()
	if strings.Contains(s, "10000") || strings.Contains(s, "Authentication error") {
		return "Permission denied (10000) — your token needs Zone:Firewall Services:Read.\n" +
			"  Edit token at dash.cloudflare.com → My Profile → API Tokens"
	}
	return s
}

func (m *fwModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
