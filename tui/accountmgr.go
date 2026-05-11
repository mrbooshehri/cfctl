package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mrbooshehri/cfctl/api"
	"github.com/mrbooshehri/cfctl/config"
	"github.com/mrbooshehri/cfctl/styles"
)

// ── view states ────────────────────────────────────────────────────────────

type acctMgrView int

const (
	acctMgrList acctMgrView = iota
	acctMgrAdd
	acctMgrEdit
	acctMgrConfirmDelete
)

// ── model ──────────────────────────────────────────────────────────────────

type accountMgrModel struct {
	view        acctMgrView
	cfg         *config.Config
	names       []string // sorted; refreshed after every mutation
	cursor      int

	// shared by add/edit forms
	nameInput   textinput.Model
	tokenInput  textinput.Model
	formFocus   int    // 0=name 1=token
	editingName string // original name (empty when adding)
	formErr     string
	formLoading bool

	width  int
	height int
}

// ── messages ───────────────────────────────────────────────────────────────

type switchAccountMsg struct{ name string } // "" means no accounts left
type closeAccountMgrMsg struct{}
type acctValidatedMsg struct {
	name, token string
	switchTo    bool // true when the app should switch to this account
}
type acctValidateErrMsg struct{ err error }

// ── constructor ────────────────────────────────────────────────────────────

func newAccountMgrModel(cfg *config.Config) accountMgrModel {
	m := accountMgrModel{cfg: cfg, names: cfg.Names()}
	for i, n := range m.names {
		if n == cfg.Active {
			m.cursor = i
			break
		}
	}
	return m
}

func (m accountMgrModel) Init() tea.Cmd { return nil }

// ── update ─────────────────────────────────────────────────────────────────

func (m accountMgrModel) Update(msg tea.Msg) (accountMgrModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case acctValidatedMsg:
		return m.commitValidated(msg)

	case acctValidateErrMsg:
		m.formLoading = false
		m.formErr = msg.err.Error()
		return m, nil
	}

	switch m.view {
	case acctMgrList:
		return m.updateList(msg)
	case acctMgrAdd, acctMgrEdit:
		return m.updateForm(msg)
	case acctMgrConfirmDelete:
		return m.updateConfirm(msg)
	}
	return m, nil
}

func (m accountMgrModel) updateList(msg tea.Msg) (accountMgrModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "j", "down":
		if m.cursor < len(m.names)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		if len(m.names) > 0 {
			m.cursor = len(m.names) - 1
		}
	case "enter":
		if len(m.names) > 0 {
			name := m.names[m.cursor]
			return m, func() tea.Msg { return switchAccountMsg{name: name} }
		}
	case "a":
		m = m.openForm("", "")
		return m, textinput.Blink
	case "e":
		if len(m.names) > 0 {
			m = m.openForm(m.names[m.cursor], "")
			return m, textinput.Blink
		}
	case "d":
		if len(m.names) > 0 {
			m.view = acctMgrConfirmDelete
		}
	case "esc", "q":
		return m, func() tea.Msg { return closeAccountMgrMsg{} }
	}
	return m, nil
}

func (m accountMgrModel) updateForm(msg tea.Msg) (accountMgrModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		if m.formFocus == 0 {
			m.nameInput, cmd = m.nameInput.Update(msg)
		} else {
			m.tokenInput, cmd = m.tokenInput.Update(msg)
		}
		return m, cmd
	}

	switch key.Type {
	case tea.KeyEsc:
		m.view = acctMgrList
		return m, nil
	case tea.KeyTab, tea.KeyShiftTab:
		m.formFocus = (m.formFocus + 1) % 2
		if m.formFocus == 0 {
			m.nameInput.Focus()
			m.tokenInput.Blur()
		} else {
			m.tokenInput.Focus()
			m.nameInput.Blur()
		}
		return m, textinput.Blink
	case tea.KeyEnter:
		if m.formFocus == 0 {
			m.formFocus = 1
			m.tokenInput.Focus()
			m.nameInput.Blur()
			return m, textinput.Blink
		}
		return m.submitForm()
	}

	var cmd tea.Cmd
	if m.formFocus == 0 {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	}
	return m, cmd
}

func (m accountMgrModel) updateConfirm(msg tea.Msg) (accountMgrModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "y", "enter":
		return m.deleteSelected()
	case "n", "esc":
		m.view = acctMgrList
	}
	return m, nil
}

// ── form helpers ───────────────────────────────────────────────────────────

func (m accountMgrModel) openForm(name, _ string) accountMgrModel {
	ni := textinput.New()
	ni.Width = 42
	ni.SetValue(name)
	ni.Placeholder = "e.g. personal, work"
	ni.Focus()

	ti := textinput.New()
	ti.Width = 42
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	if name == "" {
		ti.Placeholder = "cfut_..."
		m.view = acctMgrAdd
	} else {
		ti.Placeholder = "leave blank to keep current"
		m.view = acctMgrEdit
	}

	m.nameInput = ni
	m.tokenInput = ti
	m.formFocus = 0
	m.editingName = name
	m.formErr = ""
	m.formLoading = false
	return m
}

func (m accountMgrModel) submitForm() (accountMgrModel, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	token := strings.TrimSpace(m.tokenInput.Value())

	if name == "" {
		m.formErr = "account name cannot be empty"
		m.formFocus = 0
		m.nameInput.Focus()
		m.tokenInput.Blur()
		return m, textinput.Blink
	}

	// Duplicate name check (allow same name when editing)
	if name != m.editingName {
		for _, n := range m.names {
			if n == name {
				m.formErr = fmt.Sprintf("account %q already exists", name)
				return m, nil
			}
		}
	}

	isEdit := m.view == acctMgrEdit

	// Edit with blank token = rename only, no API call needed.
	if isEdit && token == "" {
		if name != m.editingName {
			oldToken := m.cfg.Tokens[m.editingName]
			delete(m.cfg.Tokens, m.editingName)
			m.cfg.Tokens[name] = oldToken
			if m.cfg.Active == m.editingName {
				m.cfg.Active = name
			}
			_ = config.Save(m.cfg)
			m.names = m.cfg.Names()
			m.cursor = max(0, indexOf(m.names, name))
		}
		m.view = acctMgrList
		return m, nil
	}

	if token == "" {
		m.formErr = "token cannot be empty"
		return m, nil
	}

	m.formLoading = true
	m.formErr = ""
	// Capture for the closure.
	n, t, wasActive := name, token, m.editingName == m.cfg.Active || m.editingName == ""
	return m, func() tea.Msg {
		client, err := api.New(t)
		if err != nil {
			return acctValidateErrMsg{err}
		}
		if err := client.Validate(context.Background()); err != nil {
			return acctValidateErrMsg{err}
		}
		return acctValidatedMsg{name: n, token: t, switchTo: wasActive}
	}
}

func (m accountMgrModel) commitValidated(msg acctValidatedMsg) (accountMgrModel, tea.Cmd) {
	// Remove old entry when renaming.
	if m.editingName != "" && m.editingName != msg.name {
		delete(m.cfg.Tokens, m.editingName)
		if m.cfg.Active == m.editingName {
			m.cfg.Active = msg.name
		}
	}
	m.cfg.Tokens[msg.name] = msg.token
	if m.editingName == "" {
		// New account — make it active so switchAccountMsg picks it up.
		m.cfg.Active = msg.name
	}
	_ = config.Save(m.cfg)

	m.names = m.cfg.Names()
	m.cursor = max(0, indexOf(m.names, msg.name))
	m.formLoading = false
	m.view = acctMgrList

	if msg.switchTo {
		name := msg.name
		return m, func() tea.Msg { return switchAccountMsg{name: name} }
	}
	return m, nil
}

func (m accountMgrModel) deleteSelected() (accountMgrModel, tea.Cmd) {
	if len(m.names) == 0 {
		return m, nil
	}
	name := m.names[m.cursor]
	wasActive := name == m.cfg.Active

	delete(m.cfg.Tokens, name)
	m.names = m.cfg.Names()
	if m.cursor >= len(m.names) {
		m.cursor = max(0, len(m.names)-1)
	}
	m.view = acctMgrList

	if !wasActive {
		_ = config.Save(m.cfg)
		return m, nil
	}

	// Deleted the active account.
	if len(m.names) == 0 {
		m.cfg.Active = ""
		_ = config.Save(m.cfg)
		return m, func() tea.Msg { return switchAccountMsg{name: ""} }
	}
	m.cfg.Active = m.names[m.cursor]
	_ = config.Save(m.cfg)
	next := m.cfg.Active
	return m, func() tea.Msg { return switchAccountMsg{name: next} }
}

// ── view ───────────────────────────────────────────────────────────────────

func (m accountMgrModel) View() string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}
	switch m.view {
	case acctMgrAdd:
		return m.formView(w, h, "Add Account")
	case acctMgrEdit:
		return m.formView(w, h, "Edit Account")
	case acctMgrConfirmDelete:
		return m.confirmView(w, h)
	default:
		return m.listView(w, h)
	}
}

func (m accountMgrModel) listView(w, h int) string {
	boxW := 52
	if w < boxW+6 {
		boxW = w - 6
	}

	var b strings.Builder
	if len(m.names) == 0 {
		b.WriteString(styles.DimItem.Render("No accounts yet. Press [a] to add one.") + "\n")
	} else {
		for i, name := range m.names {
			isActive := name == m.cfg.Active
			isCursor := i == m.cursor

			var badge string
			if isActive {
				badge = "  " + styles.Badge(styles.Orange).Render("active")
			}

			var line string
			switch {
			case isCursor && isActive:
				line = cursorStyle.Render("▸ "+name) + badge
			case isCursor:
				line = cursorStyle.Render("▸ " + name)
			case isActive:
				line = activeStyle.Render("  "+name) + badge
			default:
				line = styles.NormalItem.Render("  " + name)
			}
			b.WriteString(line + "\n")
		}
	}

	hint := styles.Help.Render(
		"[a] add  [e] edit  [d] delete  [enter] switch  [esc] close",
	)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		styles.SectionTitle.Render("Accounts"),
		"",
		b.String(),
		"",
		hint,
	)

	logo := lipgloss.NewStyle().Width(boxW).Align(lipgloss.Center).Render(
		styles.Title.Render("cfctl") + styles.DimItem.Render(" — Cloudflare TUI"),
	)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Orange).
		Padding(1, 3).
		Width(boxW).
		Render(inner)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, logo, "", box),
	)
}

func (m accountMgrModel) formView(w, h int, title string) string {
	boxW := 58
	if w < boxW+6 {
		boxW = w - 6
	}

	nameLabel := styles.DimItem.Render("Account name")
	tokenLabel := styles.DimItem.Render("API Token")
	if m.formFocus == 0 {
		nameLabel = styles.NormalItem.Render("Account name")
	} else {
		tokenLabel = styles.NormalItem.Render("API Token")
	}

	var status string
	switch {
	case m.formLoading:
		status = styles.DimItem.Render("Validating token…")
	case m.formErr != "":
		status = styles.Error.Render("✗ " + m.formErr)
	default:
		status = styles.Help.Render("tab · switch  enter · save  esc · back")
	}

	inner := lipgloss.JoinVertical(lipgloss.Center,
		styles.SectionTitle.Render(title),
		"",
		nameLabel,
		m.nameInput.View(),
		"",
		tokenLabel,
		m.tokenInput.View(),
		"",
		status,
	)

	logo := lipgloss.NewStyle().Width(boxW).Align(lipgloss.Center).Render(
		styles.Title.Render("cfctl") + styles.DimItem.Render(" — Cloudflare TUI"),
	)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Orange).
		Padding(1, 3).
		Width(boxW).
		Align(lipgloss.Center).
		Render(inner)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, logo, "", box),
	)
}

func (m accountMgrModel) confirmView(w, h int) string {
	name := ""
	if len(m.names) > 0 && m.cursor < len(m.names) {
		name = m.names[m.cursor]
	}

	inner := lipgloss.JoinVertical(lipgloss.Center,
		styles.SectionTitle.Render("Delete Account"),
		"",
		styles.NormalItem.Render(fmt.Sprintf(`Delete %q?`, name)),
		styles.DimItem.Render("This cannot be undone."),
		"",
		styles.Help.Render("[y / enter] confirm   [n / esc] cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Red).
		Padding(1, 4).
		Align(lipgloss.Center).
		Render(inner)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

// ── util ───────────────────────────────────────────────────────────────────

func indexOf(ss []string, s string) int {
	for i, v := range ss {
		if v == s {
			return i
		}
	}
	return -1
}
