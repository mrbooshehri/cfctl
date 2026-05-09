package styles

import "github.com/charmbracelet/lipgloss"

var (
	Orange    = lipgloss.Color("#F6821F")
	DimGray   = lipgloss.Color("#6B7280")
	White     = lipgloss.Color("#FFFFFF")
	Red       = lipgloss.Color("#EF4444")
	Green     = lipgloss.Color("#22C55E")
	Yellow    = lipgloss.Color("#EAB308")
	Blue      = lipgloss.Color("#3B82F6")
	BgDark    = lipgloss.Color("#1E1E2E")
	BgPanel   = lipgloss.Color("#252535")
	Border    = lipgloss.Color("#3D3D5C")
	Subtle    = lipgloss.Color("#4B5563")

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Orange)

	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Orange).
		Padding(0, 1)

	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(0, 1)

	ContentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(0, 1)

	ActiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Orange).
			Padding(0, 1)

	SelectedItem = lipgloss.NewStyle().
			Foreground(Orange).
			Bold(true)

	NormalItem = lipgloss.NewStyle().
			Foreground(White)

	DimItem = lipgloss.NewStyle().
		Foreground(DimGray)

	SectionTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Orange).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Border)

	Help = lipgloss.NewStyle().
		Foreground(DimGray)

	Success = lipgloss.NewStyle().
		Foreground(Green)

	Error = lipgloss.NewStyle().
		Foreground(Red)

	Badge = func(color lipgloss.Color) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(White).
			Background(color).
			Padding(0, 1)
	}

	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(Orange)

	StatusBar = lipgloss.NewStyle().
			Foreground(DimGray).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Border)
)
