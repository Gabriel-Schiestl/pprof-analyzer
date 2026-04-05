package menu

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gabri/pprof-analyzer/internal/tui/styles"
)

// NavigateTo is the message emitted when the user selects a menu item.
type NavigateTo struct{ Screen string }

type menuItem struct {
	title, desc string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

// Model is the Bubble Tea model for the main menu.
type Model struct {
	list list.Model
}

// New creates a new main menu model.
func New() Model {
	items := []list.Item{
		menuItem{"Endpoints", "Manage registered pprof endpoints"},
		menuItem{"Daemon", "Start or stop the background collector"},
		menuItem{"Dashboard", "View current status and last collection results"},
		menuItem{"Settings", "Configure Ollama and output directory"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(styles.ColorMuted)

	l := list.New(items, delegate, styles.AppWidth, 20)
	l.Title = "pprof-analyzer"
	l.Styles.Title = styles.TitleStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)

	return Model{list: l}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(menuItem); ok {
				return m, func() tea.Msg {
					return NavigateTo{Screen: item.title}
				}
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-4, msg.Height-4)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	return styles.AppStyle.Render(m.list.View())
}
