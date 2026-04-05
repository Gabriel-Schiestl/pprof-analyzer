package daemon

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gabri/pprof-analyzer/internal/app"
	"github.com/gabri/pprof-analyzer/internal/tui/styles"
)

type daemonOpResult struct{ err error }

// Model is the Bubble Tea model for the daemon control screen.
type Model struct {
	svc      *app.DaemonService
	spinner  spinner.Model
	busy     bool
	errorMsg string
}

// New creates a new daemon model.
func New(svc *app.DaemonService) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return Model{svc: svc, spinner: s}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "s":
			if !m.busy && !m.svc.IsRunning() {
				m.busy = true
				m.errorMsg = ""
				return m, tea.Batch(m.spinner.Tick, startDaemon(m.svc))
			}
		case "x":
			if !m.busy && m.svc.IsRunning() {
				m.busy = true
				m.errorMsg = ""
				return m, tea.Batch(m.spinner.Tick, stopDaemon(m.svc))
			}
		}

	case daemonOpResult:
		m.busy = false
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		}
		return m, nil

	case spinner.TickMsg:
		if m.busy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	running := m.svc.IsRunning()

	var status string
	if running {
		status = styles.StatusRunningStyle.Render("RUNNING")
	} else {
		status = styles.StatusStoppedStyle.Render("STOPPED")
	}

	content := styles.SectionStyle.Render("Daemon Control") + "\n\n" +
		"Status: " + status + "\n\n"

	if m.busy {
		content += m.spinner.View() + " Working...\n"
	} else {
		if !running {
			content += styles.HelpStyle.Render("s: start daemon")
		} else {
			content += styles.HelpStyle.Render("x: stop daemon")
		}
		content += "\n" + styles.HelpStyle.Render("ESC: back")
	}

	if m.errorMsg != "" {
		content += "\n" + styles.ErrorStyle.Render(m.errorMsg)
	}

	return styles.AppStyle.Render(content)
}

func startDaemon(svc *app.DaemonService) tea.Cmd {
	return func() tea.Msg {
		return daemonOpResult{err: svc.Start()}
	}
}

func stopDaemon(svc *app.DaemonService) tea.Cmd {
	return func() tea.Msg {
		return daemonOpResult{err: svc.Stop()}
	}
}

// BackMsg navigates back to the main menu.
type BackMsg struct{}
