package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

var (
	PrimaryColor    = lipgloss.Color("#F25D94") // Neon Pink
	SecondaryColor  = lipgloss.Color("#A550DF") // Soft Purple
	AccentColor     = lipgloss.Color("#00F5D4") // Cyan
	
	FgColor         = lipgloss.Color("#FAFAFA")
	MutedColor      = lipgloss.Color("#737373")
	BorderColor     = lipgloss.Color("#383838")
	SelectedColor   = lipgloss.Color("#2B2034") // Very dark purple for selection
	OkColor         = lipgloss.Color("#00F5D4") // Cyan for additions/ok
	WarnColor       = lipgloss.Color("#F8E16C")
	SoftRedColor    = lipgloss.Color("#FF4C8B")
	FlashColor      = lipgloss.Color("#FF0055")

	BarStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Padding(0, 1)

	AccentStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	SelectedStyle = lipgloss.NewStyle().
			Background(SelectedColor).
			Foreground(PrimaryColor).
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(PrimaryColor)

	DimStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	InlineStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	OkStyle = lipgloss.NewStyle().
			Foreground(OkColor)

	ConflictStyle = lipgloss.NewStyle().
			Foreground(WarnColor).
			Bold(true)

	AddStyle = lipgloss.NewStyle().Foreground(OkColor)
	SubStyle = lipgloss.NewStyle().Foreground(SoftRedColor)

	LeftPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(0, 1)

	RightPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor)

	ViewportStyle = lipgloss.NewStyle()

	RootStyle = lipgloss.NewStyle()

	WarnStyle = lipgloss.NewStyle().
			Foreground(FlashColor).
			Bold(true)

	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	TreemapBaseColors = []string{
		"#2B2034", "#3D2B4A", "#4F3660", "#614176", "#734C8C",
	}
)
