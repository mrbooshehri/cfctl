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

type sslModel struct {
	client  *api.Client
	zoneID  string
	packs   []cf.CertificatePack
	loading bool
	err     string
	cursor  int
}

type sslLoadedMsg struct{ packs []cf.CertificatePack }
type sslErrMsg struct{ err error }

func newSSLModel(client *api.Client, zoneID string) sslModel {
	return sslModel{client: client, zoneID: zoneID}
}

func (m sslModel) Init() tea.Cmd { return m.load() }

func (m sslModel) load() tea.Cmd {
	return func() tea.Msg {
		packs, err := m.client.ListCertificatePacks(context.Background(), m.zoneID)
		if err != nil {
			return sslErrMsg{err}
		}
		return sslLoadedMsg{packs}
	}
}

func (m sslModel) Update(msg tea.Msg) (sslModel, tea.Cmd) {
	switch msg := msg.(type) {
	case sslLoadedMsg:
		m.loading = false
		m.packs = msg.packs
		return m, nil
	case sslErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.loading = true
			return m, m.load()
		case "j", "down":
			if m.cursor < len(m.packs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}
	return m, nil
}

func statusBadge(status string) string {
	switch strings.ToLower(status) {
	case "active":
		return styles.Badge(styles.Green).Render(" active ")
	case "pending_validation", "pending_issuance":
		return styles.Badge(styles.Yellow).Render(" pending ")
	case "expired":
		return styles.Badge(styles.Red).Render(" expired ")
	default:
		return styles.Badge(styles.DimGray).Render(" " + status + " ")
	}
}

func (m sslModel) View() string {
	var b strings.Builder
	title := styles.SectionTitle.Render("SSL / TLS Certificates")
	if m.loading {
		b.WriteString(title + "  " + styles.DimItem.Render("loading...") + "\n\n")
	} else {
		b.WriteString(title + "\n\n")
	}

	if m.err != "" {
		b.WriteString(styles.Error.Render("✗ "+m.err) + "\n\n")
	}

	if len(m.packs) == 0 && !m.loading {
		b.WriteString(styles.DimItem.Render("No certificate packs found.") + "\n")
	}

	for i, p := range m.packs {
		hosts := strings.Join(p.Hosts, ", ")
		if len(hosts) > 50 {
			hosts = hosts[:47] + "..."
		}

		if i == m.cursor {
			sel := lipgloss.NewStyle().Foreground(styles.Orange).Bold(true)
			line := sel.Render("▸ "+p.Type) + "  " +
				statusBadge(p.Status) + "  " +
				sel.Render(hosts)
			b.WriteString(line + "\n")
		} else {
			line := "  " +
				styles.NormalItem.Render(fmt.Sprintf("%-12s", p.Type)) + "  " +
				statusBadge(p.Status) + "  " +
				styles.DimItem.Render(hosts)
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.Help.Render("[↑/↓] navigate  [r] refresh"))

	return b.String()
}
