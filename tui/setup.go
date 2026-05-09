package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mrbooshehri/cfctl/api"
	"github.com/mrbooshehri/cfctl/config"
	"github.com/mrbooshehri/cfctl/styles"
)

type setupModel struct {
	input   textinput.Model
	err     string
	loading bool
	width   int
	height  int
}

type tokenValidMsg struct{ client *api.Client }
type tokenErrMsg struct{ err error }

func newSetupModel() setupModel {
	ti := textinput.New()
	ti.Placeholder = "cfut_..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.Width = 50
	return setupModel{input: ti}
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
		case tea.KeyEnter:
			token := strings.TrimSpace(m.input.Value())
			if token == "" {
				m.err = "token cannot be empty"
				return m, nil
			}
			m.loading = true
			m.err = ""
			return m, validateToken(token)
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	case tokenErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
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

	var statusLine string
	if m.loading {
		statusLine = styles.DimItem.Render("Validating...")
	} else if m.err != "" {
		statusLine = styles.Error.Render("✗ " + m.err)
	} else {
		statusLine = styles.Help.Render("enter to confirm • esc to quit")
	}

	inner := lipgloss.JoinVertical(lipgloss.Center,
		styles.SectionTitle.Render("Enter your Cloudflare API Token"),
		"",
		styles.DimItem.Render("Tokens start with cfut_ and can be created at"),
		styles.DimItem.Render("dash.cloudflare.com → My Profile → API Tokens"),
		"",
		m.input.View(),
		"",
		statusLine,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Orange).
		Padding(1, 3).
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(inner)

	modal := lipgloss.JoinVertical(lipgloss.Center, logo, "", box)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modal)
}

func validateToken(token string) tea.Cmd {
	return func() tea.Msg {
		client, err := api.New(token)
		if err != nil {
			return tokenErrMsg{err}
		}
		if err := client.Validate(context.Background()); err != nil {
			return tokenErrMsg{err}
		}
		if err := config.Save(&config.Config{Token: token}); err != nil {
			return tokenErrMsg{err}
		}
		return tokenValidMsg{client}
	}
}
