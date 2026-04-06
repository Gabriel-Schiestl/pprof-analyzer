package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/app"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/config"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui/daemon"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui/dashboard"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui/endpoints"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui/menu"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui/settings"
)

type screen int

const (
	screenMenu screen = iota
	screenEndpoints
	screenEndpointForm
	screenEndpointConfirm
	screenDaemon
	screenDashboard
	screenSettings
)

// AppModel is the root Bubble Tea model that manages navigation between screens.
type AppModel struct {
	current screen

	// sub-models
	menu            menu.Model
	endpointList    endpoints.ListModel
	endpointForm    endpoints.FormModel
	endpointConfirm endpoints.ConfirmModel
	daemonView      daemon.Model
	dashboardView   dashboard.Model
	settingsView    settings.Model

	// services
	endpointSvc *app.EndpointService
	daemonSvc   *app.DaemonService
	metadata    app.MetadataStore
	cfg         *config.Config
}

// New creates the root AppModel and wires all sub-models.
func New(
	endpointSvc *app.EndpointService,
	daemonSvc *app.DaemonService,
	endpointRepo app.EndpointRepository,
	metadata app.MetadataStore,
	cfg *config.Config,
) AppModel {
	return AppModel{
		current:       screenMenu,
		menu:          menu.New(),
		endpointList:  endpoints.NewListModel(endpointSvc),
		daemonView:    daemon.New(daemonSvc),
		dashboardView: dashboard.New(daemonSvc, endpointRepo, metadata),
		settingsView:  settings.New(cfg),
		endpointSvc:   endpointSvc,
		daemonSvc:     daemonSvc,
		metadata:      metadata,
		cfg:           cfg,
	}
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return m.menu.Init()
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global quit
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	// Navigation messages from sub-models are handled here before routing to
	// the current screen, so commands are never executed speculatively.
	switch msg := msg.(type) {
	case menu.NavigateTo:
		return m.handleMenuNav(msg.Screen)

	case endpoints.ShowFormMsg:
		m.endpointForm = endpoints.NewFormModel(m.endpointSvc, msg.Endpoint)
		m.current = screenEndpointForm
		return m, m.endpointForm.Init()

	case endpoints.ShowConfirmMsg:
		m.endpointConfirm = endpoints.NewConfirmModel(m.endpointSvc, msg.Endpoint)
		m.current = screenEndpointConfirm
		return m, nil

	case endpoints.BackMsg:
		if m.current == screenEndpoints {
			m.current = screenMenu
		} else {
			m.current = screenEndpoints
			m.endpointList = endpoints.NewListModel(m.endpointSvc)
		}
		return m, nil

	case endpoints.FormSubmittedMsg:
		m.current = screenEndpoints
		m.endpointList = endpoints.NewListModel(m.endpointSvc)
		return m, nil

	case daemon.BackMsg:
		m.current = screenMenu
		return m, nil

	case dashboard.BackMsg:
		m.current = screenMenu
		return m, nil

	case settings.BackMsg, settings.SavedMsg:
		m.current = screenMenu
		return m, nil
	}

	// Route to the active screen.
	switch m.current {
	case screenMenu:
		newMenu, cmd := m.menu.Update(msg)
		m.menu = newMenu
		return m, cmd
	case screenEndpoints:
		newList, cmd := m.endpointList.Update(msg)
		m.endpointList = newList
		return m, cmd
	case screenEndpointForm:
		newForm, cmd := m.endpointForm.Update(msg)
		m.endpointForm = newForm
		return m, cmd
	case screenEndpointConfirm:
		newConfirm, cmd := m.endpointConfirm.Update(msg)
		m.endpointConfirm = newConfirm
		return m, cmd
	case screenDaemon:
		newDaemon, cmd := m.daemonView.Update(msg)
		m.daemonView = newDaemon
		return m, cmd
	case screenDashboard:
		newDash, cmd := m.dashboardView.Update(msg)
		m.dashboardView = newDash
		return m, cmd
	case screenSettings:
		newSettings, cmd := m.settingsView.Update(msg)
		m.settingsView = newSettings
		return m, cmd
	}
	return m, nil
}

// View implements tea.Model.
func (m AppModel) View() string {
	switch m.current {
	case screenMenu:
		return m.menu.View()
	case screenEndpoints:
		return m.endpointList.View()
	case screenEndpointForm:
		return m.endpointForm.View()
	case screenEndpointConfirm:
		return m.endpointConfirm.View()
	case screenDaemon:
		return m.daemonView.View()
	case screenDashboard:
		return m.dashboardView.View()
	case screenSettings:
		return m.settingsView.View()
	}
	return ""
}

func (m AppModel) handleMenuNav(screen string) (tea.Model, tea.Cmd) {
	switch screen {
	case "Endpoints":
		m.current = screenEndpoints
		m.endpointList = endpoints.NewListModel(m.endpointSvc)
	case "Daemon":
		m.current = screenDaemon
	case "Dashboard":
		m.current = screenDashboard
		return m, m.dashboardView.Init()
	case "Settings":
		m.current = screenSettings
	}
	return m, nil
}
