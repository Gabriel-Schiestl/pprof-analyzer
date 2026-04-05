package dashboard

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gabri/pprof-analyzer/internal/app"
	"github.com/gabri/pprof-analyzer/internal/domain"
	"github.com/gabri/pprof-analyzer/internal/tui/styles"
)

const refreshInterval = 5 * time.Second

type refreshMsg struct{}

// BackMsg navigates back to the main menu.
type BackMsg struct{}

// Model is the Bubble Tea model for the dashboard view.
type Model struct {
	table        table.Model
	daemonSvc    *app.DaemonService
	endpointRepo app.EndpointRepository
	metadata     app.MetadataStore
	errorMsg     string
}

// New creates a new dashboard model.
func New(daemonSvc *app.DaemonService, endpointRepo app.EndpointRepository, metadata app.MetadataStore) Model {
	cols := []table.Column{
		{Title: "Application", Width: 20},
		{Title: "Environment", Width: 12},
		{Title: "Last Collection", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Alerts", Width: 8},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithHeight(12),
	)

	s := table.DefaultStyles()
	s.Header = styles.TableHeaderStyle
	s.Selected = styles.SelectedRowStyle
	t.SetStyles(s)

	m := Model{
		table:        t,
		daemonSvc:    daemonSvc,
		endpointRepo: endpointRepo,
		metadata:     metadata,
	}
	m.refresh()
	return m
}

// Init implements tea.Model — starts the auto-refresh ticker.
func (m Model) Init() tea.Cmd {
	return tea.Tick(refreshInterval, func(_ time.Time) tea.Msg {
		return refreshMsg{}
	})
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "r":
			m.refresh()
			return m, nil
		}
	case refreshMsg:
		m.refresh()
		return m, tea.Tick(refreshInterval, func(_ time.Time) tea.Msg {
			return refreshMsg{}
		})
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width - 4)
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	running := m.daemonSvc.IsRunning()
	var daemonStatus string
	if running {
		daemonStatus = styles.StatusRunningStyle.Render("Daemon: RUNNING")
	} else {
		daemonStatus = styles.StatusStoppedStyle.Render("Daemon: STOPPED")
	}

	header := styles.SectionStyle.Render("Dashboard")
	help := styles.HelpStyle.Render("r: refresh  ESC: back")

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		daemonStatus,
		"",
		m.table.View(),
		"",
		help,
	)

	if m.errorMsg != "" {
		content += "\n" + styles.ErrorStyle.Render(m.errorMsg)
	}

	return styles.AppStyle.Render(content)
}

func (m *Model) refresh() {
	endpoints, err := m.endpointRepo.List()
	if err != nil {
		m.errorMsg = err.Error()
		return
	}

	rows := make([]table.Row, len(endpoints))
	for i, ep := range endpoints {
		lastCollection := "never"
		alertCount := "—"
		status := "no data"

		if run, err := m.metadata.GetLastRun(ep.ID); err == nil && run != nil {
			lastCollection = run.CompletedAt.Format("2006-01-02 15:04")
			status = string(run.Status)
			alertCount = countAlerts(run)
		}

		rows[i] = table.Row{
			ep.Name,
			string(ep.Environment),
			lastCollection,
			status,
			alertCount,
		}
	}
	m.table.SetRows(rows)
}

func countAlerts(run *domain.CollectionRun) string {
	if run.Status == domain.RunStatusFailed {
		return "ERR"
	}
	return "0"
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return fmt.Sprintf("%s ago", time.Since(t).Round(time.Second))
}
