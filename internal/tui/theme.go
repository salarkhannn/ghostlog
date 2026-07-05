package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

var (
	BgColor         = lipgloss.Color("#0e0d0c")
	FgColor         = lipgloss.Color("#e8e3dc")
	MutedColor      = lipgloss.Color("#7a7168")
	BorderColor     = lipgloss.Color("#26221d")
	SelectedColor   = lipgloss.Color("#1e1a16")
	ActiveColor     = lipgloss.Color("#c17a4f")
	OkColor         = lipgloss.Color("#6f9c72")
	WarnColor       = lipgloss.Color("#b8963f")
	SoftRedColor    = lipgloss.Color("#b85c52")
	FlashColor      = lipgloss.Color("#ff2d78") // Reserved ONLY for treemap flash and secret-leak warnings

	BarStyle = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(MutedColor).
			Padding(0, 1)

	AccentStyle = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(ActiveColor).
			Bold(true)

	SelectedStyle = lipgloss.NewStyle().
			Background(SelectedColor).
			Foreground(FgColor).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(MutedColor)

	InlineStyle = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(MutedColor)

	OkStyle = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(OkColor)

	ConflictStyle = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(WarnColor).
			Bold(true)

	AddStyle = lipgloss.NewStyle().Background(BgColor).Foreground(OkColor)
	SubStyle = lipgloss.NewStyle().Background(BgColor).Foreground(SoftRedColor)

	LeftPaneStyle = lipgloss.NewStyle().
			Background(BgColor).
			Padding(0, 1)

	RightPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(BorderColor).
			BorderBackground(BgColor).
			Background(BgColor)

	ViewportStyle = lipgloss.NewStyle().
			Background(BgColor)

	RootStyle = lipgloss.NewStyle().
			Background(BgColor)

	WarnStyle = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(FlashColor).
			Bold(true)

	BreadcrumbStyle = lipgloss.NewStyle().
			Background(BgColor).
			Foreground(ActiveColor).
			Bold(true)

	TreemapBaseColors = []string{
		"#3b332b", "#2e2720", "#473c33", "#251e18", "#352c24",
	}
)
