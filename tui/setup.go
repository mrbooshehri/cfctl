package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mrbooshehri/cfctl/api"
	"github.com/mrbooshehri/cfctl/styles"
)

type setupModel struct {
	nameInput  textinput.Model
	tokenInput textinput.Model
	focusIdx   int // 0=name 1=token
	err        string
	loading    bool
	width      int
	height     int
}

type tokenValidMsg struct {
	name   string
	token  string
	client *api.Client
}
type tokenErrMsg struct{ err error }

func newSetupModel() setupModel {
	ni := textinput.New()
	ni.Placeholder = "e.g. personal, work"
	ni.Width = 50
	ni.Focus()

	ti := textinput.New()
	ti.Placeholder = "cfut_..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Width = 50

	return setupModel{nameInput: ni, tokenInput: ti}
}

func (m setupModel) Init() tea.Cmd { return textinput.Blink }

func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyTab, tea.KeyShiftTab:
			m.focusIdx = (m.focusIdx + 1) % 2
			if m.focusIdx == 0 {
				m.nameInput.Focus()
				m.tokenInput.Blur()
			} else {
				m.tokenInput.Focus()
				m.nameInput.Blur()
			}
			return m, textinput.Blink
		case tea.KeyEnter:
			if m.focusIdx == 0 {
				m.focusIdx = 1
				m.tokenInput.Focus()
				m.nameInput.Blur()
				return m, textinput.Blink
			}
			name := strings.TrimSpace(m.nameInput.Value())
			token := strings.TrimSpace(m.tokenInput.Value())
			if name == "" {
				m.err = "account name cannot be empty"
				m.focusIdx = 0
				m.nameInput.Focus()
				m.tokenInput.Blur()
				return m, textinput.Blink
			}
			if token == "" {
				m.err = "token cannot be empty"
				return m, nil
			}
			m.loading = true
			m.err = ""
			return m, validateToken(name, token)
		}

	case tokenErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil
	}

	var cmd tea.Cmd
	if m.focusIdx == 0 {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	}
	return m, cmd
}

func (m setupModel) View() string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	boxWidth := 58
	if w < boxWidth+6 {
		boxWidth = w - 6
	}

	center := lipgloss.NewStyle().Width(boxWidth).Align(lipgloss.Center)
	logo := center.Render(
		styles.Title.Render("cfctl") + styles.DimItem.Render(" — Cloudflare TUI"),
	)

	var status string
	switch {
	case m.loading:
		status = styles.DimItem.Render("Validating…")
	case m.err != "":
		status = styles.Error.Render("✗ " + m.err)
	default:
		status = styles.Help.Render("tab · switch fields   enter · confirm   esc · quit")
	}

	nameLabel := styles.DimItem.Render("Account name")
	tokenLabel := styles.DimItem.Render("API Token")
	if m.focusIdx == 0 {
		nameLabel = styles.NormalItem.Render("Account name")
	} else {
		tokenLabel = styles.NormalItem.Render("API Token")
	}

	inner := lipgloss.JoinVertical(lipgloss.Center,
		styles.SectionTitle.Render("Add your Cloudflare Account"),
		"",
		styles.DimItem.Render("Tokens start with cfut_ and can be created at"),
		styles.DimItem.Render("dash.cloudflare.com → My Profile → API Tokens"),
		"",
		nameLabel,
		m.nameInput.View(),
		"",
		tokenLabel,
		m.tokenInput.View(),
		"",
		status,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Orange).
		Padding(1, 3).
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(inner)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, logo, "", box),
	)
}

func validateToken(name, token string) tea.Cmd {
	return func() tea.Msg {
		client, err := api.New(token)
		if err != nil {
			return tokenErrMsg{err}
		}
		if err := client.Validate(context.Background()); err != nil {
			return tokenErrMsg{err}
		}
		return tokenValidMsg{name: name, token: token, client: client}
	}
}
