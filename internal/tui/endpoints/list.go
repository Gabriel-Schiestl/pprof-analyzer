package endpoints

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/app"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui/styles"
)

// --- Navigation messages ---

type ShowFormMsg struct{ Endpoint *domain.Endpoint } // nil = add mode
type ShowConfirmMsg struct{ Endpoint domain.Endpoint }
type BackMsg struct{}

// ListModel is the Bubble Tea model for the endpoint list view.
type ListModel struct {
	table    table.Model
	svc      *app.EndpointService
	items    []domain.Endpoint
	errorMsg string
}

// NewListModel creates a new list model.
func NewListModel(svc *app.EndpointService) ListModel {
	cols := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "URL", Width: 30},
		{Title: "Environment", Width: 14},
		{Title: "Interval", Width: 10},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	s := table.DefaultStyles()
	s.Header = styles.TableHeaderStyle
	s.Selected = styles.SelectedRowStyle
	t.SetStyles(s)

	m := ListModel{
		table: t,
		svc:   svc,
	}
	m.reload()
	return m
}

func (m *ListModel) reload() {
	endpoints, err := m.svc.List()
	if err != nil {
		m.errorMsg = err.Error()
		return
	}
	m.items = endpoints

	rows := make([]table.Row, len(endpoints))
	for i, ep := range endpoints {
		rows[i] = table.Row{
			ep.Name,
			ep.BaseURL,
			string(ep.Environment),
			formatDuration(ep.CollectInterval),
		}
	}
	m.table.SetRows(rows)
}

// Init implements tea.Model.
func (m ListModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "a":
			return m, func() tea.Msg { return ShowFormMsg{} }
		case "e":
			if len(m.items) > 0 {
				ep := m.items[m.table.Cursor()]
				return m, func() tea.Msg { return ShowFormMsg{Endpoint: &ep} }
			}
		case "d":
			if len(m.items) > 0 {
				ep := m.items[m.table.Cursor()]
				return m, func() tea.Msg { return ShowConfirmMsg{Endpoint: ep} }
			}
		}
	case ReloadMsg:
		m.reload()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m ListModel) View() string {
	header := styles.SectionStyle.Render("Endpoints")
	help := styles.HelpStyle.Render("a: add  e: edit  d: delete  ESC: back")

	content := header + "\n" + m.table.View()
	if m.errorMsg != "" {
		content += "\n" + styles.ErrorStyle.Render(m.errorMsg)
	}

	return styles.AppStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, content, help),
	)
}

// ReloadMsg signals the list to refresh its data.
type ReloadMsg struct{}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	secs := int(d.Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm", secs/60)
}
