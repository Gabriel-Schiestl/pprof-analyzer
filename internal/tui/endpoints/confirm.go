package endpoints

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gabri/pprof-analyzer/internal/app"
	"github.com/gabri/pprof-analyzer/internal/domain"
	"github.com/gabri/pprof-analyzer/internal/tui/styles"
)

// ConfirmModel is the deletion confirmation dialog.
type ConfirmModel struct {
	endpoint domain.Endpoint
	svc      *app.EndpointService
	errorMsg string
}

// NewConfirmModel creates a confirm model for the given endpoint.
func NewConfirmModel(svc *app.EndpointService, ep domain.Endpoint) ConfirmModel {
	return ConfirmModel{endpoint: ep, svc: svc}
}

// Init implements tea.Model.
func (m ConfirmModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			if err := m.svc.Delete(m.endpoint.ID); err != nil {
				m.errorMsg = err.Error()
				return m, nil
			}
			return m, func() tea.Msg { return FormSubmittedMsg{} }
		case "n", "N", "esc":
			return m, func() tea.Msg { return BackMsg{} }
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m ConfirmModel) View() string {
	msg := fmt.Sprintf("Delete endpoint %q? (y/n)", m.endpoint.Name)
	out := styles.SectionStyle.Render("Confirm Deletion") + "\n\n" +
		styles.NormalStyle.Render(msg)

	if m.errorMsg != "" {
		out += "\n" + styles.ErrorStyle.Render(m.errorMsg)
	}

	out += "\n" + styles.HelpStyle.Render("y: confirm  n/ESC: cancel")
	return styles.AppStyle.Render(out)
}
