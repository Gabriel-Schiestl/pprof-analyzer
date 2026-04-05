package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	ColorPrimary   = lipgloss.Color("#3B82F6") // blue
	ColorSecondary = lipgloss.Color("#8B5CF6") // purple
	ColorSuccess   = lipgloss.Color("#16A34A") // green
	ColorWarning   = lipgloss.Color("#D97706") // amber
	ColorError     = lipgloss.Color("#DC2626") // red
	ColorMuted     = lipgloss.Color("#6B7280") // gray
	ColorDark      = lipgloss.Color("#1F2937") // near-black
	ColorLightBg   = lipgloss.Color("#F3F4F6") // light gray

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Section header
	SectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorDark).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorMuted).
			MarginBottom(1)

	// Normal text
	NormalStyle = lipgloss.NewStyle().
			Foreground(ColorDark)

	// Muted text
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Status: running
	StatusRunningStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSuccess).
				PaddingLeft(1).
				PaddingRight(1)

	// Status: stopped
	StatusStoppedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorError).
				PaddingLeft(1).
				PaddingRight(1)

	// Severity badges
	BadgeCritical = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#DC2626")).
			PaddingLeft(1).
			PaddingRight(1)

	BadgeWarning = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#D97706")).
			PaddingLeft(1).
			PaddingRight(1)

	BadgeNormal = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#16A34A")).
			PaddingLeft(1).
			PaddingRight(1)

	// Table header
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorMuted)

	// Selected row
	SelectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(ColorPrimary)

	// App container
	AppStyle = lipgloss.NewStyle().
			Margin(1, 2)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	// Error message
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)
)

// AppWidth is the default content width.
const AppWidth = 80
