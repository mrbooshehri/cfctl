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

type dnsModel struct {
	client      *api.Client
	log         *logger.Logger
	zoneID      string
	zoneName    string
	records     []cf.DNSRecord
	cursor      int
	offset      int
	form        dnsForm
	showForm    bool
	editID      string
	showConfirm bool
	confirmIdx  int
	loading     bool
	err         string
	statusMsg   string
	width       int
	height      int
}

type dnsForm struct {
	fields  []textinput.Model
	focused int
	isEdit  bool
	proxied bool
}

type dnsLoadedMsg struct{ records []cf.DNSRecord }
type dnsErrMsg struct{ err error }
type dnsOKMsg struct{ msg string }

func newDNSModel(client *api.Client, log *logger.Logger, zoneID, zoneName string) dnsModel {
	return dnsModel{client: client, log: log, zoneID: zoneID, zoneName: zoneName}
}

func (m dnsModel) Init() tea.Cmd { return m.loadRecords() }

func (m dnsModel) loadRecords() tea.Cmd {
	client, log, zoneID, zoneName := m.client, m.log, m.zoneID, m.zoneName
	return func() tea.Msg {
		records, err := client.ListDNSRecords(context.Background(), zoneID)
		if err != nil {
			if log != nil {
				log.Write(logger.LevelError, "DNS", zoneName, "List records: "+err.Error())
			}
			return dnsErrMsg{err}
		}
		if log != nil {
			log.Write(logger.LevelInfo, "DNS", zoneName, fmt.Sprintf("Listed %d records", len(records)))
		}
		return dnsLoadedMsg{records}
	}
}

func (m dnsModel) visibleHeight() int {
	h := m.height - 11
	if h < 3 {
		h = 3
	}
	return h
}

func (m *dnsModel) scrollToCursor() {
	vh := m.visibleHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+vh {
		m.offset = m.cursor - vh + 1
	}
}

func (m dnsModel) Update(msg tea.Msg) (dnsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case dnsLoadedMsg:
		m.loading = false
		m.records = msg.records
		m.cursor = 0
		m.offset = 0
		return m, nil

	case dnsErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil

	case dnsOKMsg:
		m.statusMsg = msg.msg
		m.showForm = false
		return m, m.loadRecords()

	case tea.KeyMsg:
		if m.showForm {
			return m.updateForm(msg)
		}
		if m.showConfirm {
			return m.updateConfirm(msg)
		}
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.records)-1 {
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
			if len(m.records) > 0 {
				m.cursor = len(m.records) - 1
				m.scrollToCursor()
			}
		case "n":
			m.form = newDNSForm(nil)
			m.showForm = true
			m.err = ""
			m.statusMsg = ""
		case "e":
			if len(m.records) > 0 && m.cursor < len(m.records) {
				m.form = newDNSForm(&m.records[m.cursor])
				m.editID = m.records[m.cursor].ID
				m.showForm = true
				m.err = ""
				m.statusMsg = ""
			}
		case "d":
			if len(m.records) > 0 && m.cursor < len(m.records) {
				m.showConfirm = true
				m.confirmIdx = m.cursor
				m.err = ""
				m.statusMsg = ""
			}
		case "r":
			m.loading = true
			return m, m.loadRecords()
		}
	}
	return m, nil
}

func newDNSForm(record *cf.DNSRecord) dnsForm {
	placeholders := []string{"Type (A/AAAA/CNAME/MX/TXT)", "Name", "Content", "TTL (1=Auto)"}
	fields := make([]textinput.Model, len(placeholders))
	for i, p := range placeholders {
		ti := textinput.New()
		ti.Placeholder = p
		ti.Width = 45
		fields[i] = ti
	}
	isEdit := false
	proxied := false
	if record != nil {
		isEdit = true
		fields[0].SetValue(record.Type)
		fields[1].SetValue(record.Name)
		fields[2].SetValue(record.Content)
		fields[3].SetValue(fmt.Sprintf("%d", record.TTL))
		if record.Proxied != nil {
			proxied = *record.Proxied
		}
	}
	fields[0].Focus()
	return dnsForm{fields: fields, focused: 0, isEdit: isEdit, proxied: proxied}
}

func (m dnsModel) updateForm(msg tea.KeyMsg) (dnsModel, tea.Cmd) {
	total := len(m.form.fields) + 1  // 4 text fields + 1 proxy toggle
	proxyIdx := len(m.form.fields)   // index 4

	onTextField := m.form.focused < len(m.form.fields)

	switch msg.Type {
	case tea.KeyEsc:
		m.showForm = false
		return m, nil

	case tea.KeyTab, tea.KeyDown:
		if onTextField {
			m.form.fields[m.form.focused].Blur()
		}
		m.form.focused = (m.form.focused + 1) % total
		if m.form.focused < len(m.form.fields) {
			m.form.fields[m.form.focused].Focus()
			return m, textinput.Blink
		}
		return m, nil

	case tea.KeyShiftTab, tea.KeyUp:
		if onTextField {
			m.form.fields[m.form.focused].Blur()
		}
		m.form.focused = (m.form.focused - 1 + total) % total
		if m.form.focused < len(m.form.fields) {
			m.form.fields[m.form.focused].Focus()
			return m, textinput.Blink
		}
		return m, nil

	case tea.KeyEnter:
		if m.form.focused < proxyIdx {
			m.form.fields[m.form.focused].Blur()
			m.form.focused++
			if m.form.focused < len(m.form.fields) {
				m.form.fields[m.form.focused].Focus()
				return m, textinput.Blink
			}
			return m, nil // landed on proxy toggle
		}
		// Enter on proxy toggle → submit
		return m.submitForm()
	}

	// Space toggles proxy when on the proxy toggle field
	if msg.String() == " " && m.form.focused == proxyIdx {
		m.form.proxied = !m.form.proxied
		return m, nil
	}

	// Forward to focused text input
	if onTextField {
		var cmd tea.Cmd
		m.form.fields[m.form.focused], cmd = m.form.fields[m.form.focused].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m dnsModel) submitForm() (dnsModel, tea.Cmd) {
	recType := strings.ToUpper(strings.TrimSpace(m.form.fields[0].Value()))
	name := strings.TrimSpace(m.form.fields[1].Value())
	content := strings.TrimSpace(m.form.fields[2].Value())
	ttlStr := strings.TrimSpace(m.form.fields[3].Value())

	if recType == "" || name == "" || content == "" {
		m.err = "type, name, and content are required"
		return m, nil
	}

	ttl := 1
	if ttlStr != "" && ttlStr != "1" && ttlStr != "Auto" {
		fmt.Sscanf(ttlStr, "%d", &ttl)
	}

	client := m.client
	zoneID := m.zoneID

	proxied := m.form.proxied
	log, zoneName := m.log, m.zoneName

	if m.form.isEdit {
		editID := m.editID
		params := cf.UpdateDNSRecordParams{
			ID: editID, Type: recType, Name: name, Content: content, TTL: ttl, Proxied: &proxied,
		}
		return m, func() tea.Msg {
			if _, err := client.UpdateDNSRecord(context.Background(), zoneID, params); err != nil {
				if log != nil {
					log.Write(logger.LevelError, "DNS", zoneName, "Update record: "+err.Error())
				}
				return dnsErrMsg{err}
			}
			if log != nil {
				log.Write(logger.LevelSuccess, "DNS", zoneName,
					fmt.Sprintf("Updated %s record %s → %s", recType, name, content))
			}
			return dnsOKMsg{"Record updated"}
		}
	}

	params := cf.CreateDNSRecordParams{
		Type: recType, Name: name, Content: content, TTL: ttl, Proxied: &proxied,
	}
	return m, func() tea.Msg {
		if _, err := client.CreateDNSRecord(context.Background(), zoneID, params); err != nil {
			if log != nil {
				log.Write(logger.LevelError, "DNS", zoneName, "Create record: "+err.Error())
			}
			return dnsErrMsg{err}
		}
		if log != nil {
			log.Write(logger.LevelSuccess, "DNS", zoneName,
				fmt.Sprintf("Created %s record %s → %s", recType, name, content))
		}
		return dnsOKMsg{"Record created"}
	}
}

func (m dnsModel) View() string {
	if m.showForm {
		return m.formView()
	}
	if m.showConfirm {
		return m.confirmView()
	}

	var b strings.Builder

	title := styles.SectionTitle.Render("DNS Records")
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
	b.WriteString(styles.Help.Render("[n] new  [e] edit  [d] delete  [r] refresh  [j/k] navigate  [g/G] top/bottom"))

	return b.String()
}

func (m dnsModel) renderTable() string {
	if len(m.records) == 0 {
		return styles.DimItem.Render("  No DNS records found.") + "\n"
	}

	// Column widths
	available := m.width - 8 // margins
	if available < 40 {
		available = 40
	}
	typeW := 7
	ttlW := 6
	proxyW := 5
	nameW := (available - typeW - ttlW - proxyW) * 2 / 5
	contentW := available - typeW - ttlW - proxyW - nameW

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.TableHeader.Width(typeW).Render("TYPE"),
		styles.TableHeader.Width(nameW).Render("NAME"),
		styles.TableHeader.Width(contentW).Render("CONTENT"),
		styles.TableHeader.Width(ttlW).Render("TTL"),
		styles.TableHeader.Width(proxyW).Render("PROXY"),
	)

	var b strings.Builder
	b.WriteString("  " + header + "\n")
	b.WriteString("  " + styles.DimItem.Render(strings.Repeat("─", available)) + "\n")

	vh := m.visibleHeight()
	end := m.offset + vh
	if end > len(m.records) {
		end = len(m.records)
	}

	sel := lipgloss.NewStyle().Foreground(styles.Orange).Bold(true)
	normal := lipgloss.NewStyle().Foreground(styles.White)
	dim := lipgloss.NewStyle().Foreground(styles.DimGray)

	for i := m.offset; i < end; i++ {
		r := m.records[i]
		isSelected := i == m.cursor

		ttl := fmt.Sprintf("%d", r.TTL)
		if r.TTL == 1 {
			ttl = "Auto"
		}
		proxy := "✗"
		if r.Proxied != nil && *r.Proxied {
			proxy = "✓"
		}

		name := runesTrunc(r.Name, nameW-1)
		content := runesTrunc(r.Content, contentW-1)

		if isSelected {
			row := "▸ " + sel.Width(typeW).Render(r.Type) +
				sel.Width(nameW).Render(name) +
				sel.Width(contentW).Render(content) +
				sel.Width(ttlW).Render(ttl) +
				sel.Width(proxyW).Render(proxy)
			b.WriteString(row + "\n")
		} else {
			row := "  " + normal.Width(typeW).Render(r.Type) +
				dim.Width(nameW).Render(name) +
				dim.Width(contentW).Render(content) +
				dim.Width(ttlW).Render(ttl) +
				dim.Width(proxyW).Render(proxy)
			b.WriteString(row + "\n")
		}
	}

	if len(m.records) > vh {
		b.WriteString(styles.DimItem.Render(fmt.Sprintf(
			"  %d-%d of %d", m.offset+1, end, len(m.records))) + "\n")
	}

	return b.String()
}

func (m dnsModel) formView() string {
	title := "New DNS Record"
	if m.form.isEdit {
		title = "Edit DNS Record"
	}
	proxyIdx := len(m.form.fields)

	var b strings.Builder
	b.WriteString(styles.SectionTitle.Render(title) + "\n\n")

	for i, f := range m.form.fields {
		if i == m.form.focused {
			b.WriteString(styles.SelectedItem.Render("> ") + f.View() + "\n")
		} else {
			b.WriteString("  " + f.View() + "\n")
		}
	}

	// Proxy toggle
	proxyLabel := "[ ] Not proxied (Cloudflare DNS only)"
	if m.form.proxied {
		proxyLabel = "[✓] Proxied (traffic routed through Cloudflare)"
	}
	if m.form.focused == proxyIdx {
		b.WriteString(styles.SelectedItem.Render("> "+proxyLabel) +
			styles.Help.Render("  [space] toggle") + "\n")
	} else {
		if m.form.proxied {
			b.WriteString("  " + styles.Success.Render(proxyLabel) + "\n")
		} else {
			b.WriteString("  " + styles.NormalItem.Render(proxyLabel) + "\n")
		}
	}

	b.WriteString("\n")
	if m.err != "" {
		b.WriteString(styles.Error.Render("✗ "+m.err) + "\n\n")
	}
	b.WriteString(styles.Help.Render("[tab/↑↓] navigate  [space] toggle proxy  [enter] next/submit  [esc] cancel"))
	return b.String()
}

func (m dnsModel) updateConfirm(msg tea.KeyMsg) (dnsModel, tea.Cmd) {
	switch {
	case msg.String() == "y" || msg.String() == "Y":
		if m.confirmIdx >= len(m.records) {
			m.showConfirm = false
			return m, nil
		}
		r := m.records[m.confirmIdx]
		client, log, zoneID, zoneName := m.client, m.log, m.zoneID, m.zoneName
		m.showConfirm = false
		m.loading = true
		return m, func() tea.Msg {
			if err := client.DeleteDNSRecord(context.Background(), zoneID, r.ID); err != nil {
				if log != nil {
					log.Write(logger.LevelError, "DNS", zoneName, "Delete record: "+err.Error())
				}
				return dnsErrMsg{err}
			}
			if log != nil {
				log.Write(logger.LevelSuccess, "DNS", zoneName,
					fmt.Sprintf("Deleted %s record %s → %s", r.Type, r.Name, r.Content))
			}
			return dnsOKMsg{"Record deleted"}
		}
	default:
		m.showConfirm = false
	}
	return m, nil
}

func (m dnsModel) confirmView() string {
	if m.confirmIdx >= len(m.records) {
		return ""
	}
	r := m.records[m.confirmIdx]

	ttl := fmt.Sprintf("%d", r.TTL)
	if r.TTL == 1 {
		ttl = "Auto"
	}
	proxy := "no"
	if r.Proxied != nil && *r.Proxied {
		proxy = "yes"
	}

	var b strings.Builder
	b.WriteString(styles.Error.Render("Delete DNS Record?") + "\n\n")
	b.WriteString(styles.DimItem.Render("  This action cannot be undone.") + "\n\n")
	b.WriteString("  " + styles.DimItem.Render("Type:    ") + styles.NormalItem.Render(r.Type) + "\n")
	b.WriteString("  " + styles.DimItem.Render("Name:    ") + styles.NormalItem.Render(r.Name) + "\n")
	b.WriteString("  " + styles.DimItem.Render("Content: ") + styles.NormalItem.Render(r.Content) + "\n")
	b.WriteString("  " + styles.DimItem.Render("TTL:     ") + styles.NormalItem.Render(ttl) + "\n")
	b.WriteString("  " + styles.DimItem.Render("Proxied: ") + styles.NormalItem.Render(proxy) + "\n")
	b.WriteString("\n")
	b.WriteString(styles.Error.Render("[y] confirm delete") + "  " + styles.Help.Render("[any other key] cancel"))
	return b.String()
}

func (m *dnsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// runesTrunc truncates s to at most n runes, adding "…" if truncated.
func runesTrunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
