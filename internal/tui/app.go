package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gabri/pprof-analyzer/internal/app"
	"github.com/gabri/pprof-analyzer/internal/config"
	"github.com/gabri/pprof-analyzer/internal/tui/daemon"
	"github.com/gabri/pprof-analyzer/internal/tui/dashboard"
	"github.com/gabri/pprof-analyzer/internal/tui/endpoints"
	"github.com/gabri/pprof-analyzer/internal/tui/menu"
	"github.com/gabri/pprof-analyzer/internal/tui/settings"
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
	menu          menu.Model
	endpointList  endpoints.ListModel
	endpointForm  endpoints.FormModel
	endpointConfirm endpoints.ConfirmModel
	daemonView    daemon.Model
	dashboardView dashboard.Model
	settingsView  settings.Model

	// services
	endpointSvc  *app.EndpointService
	daemonSvc    *app.DaemonService
	metadata     app.MetadataStore
	cfg          *config.Config
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

	switch m.current {
	case screenMenu:
		return m.updateMenu(msg)
	case screenEndpoints:
		return m.updateEndpointList(msg)
	case screenEndpointForm:
		return m.updateEndpointForm(msg)
	case screenEndpointConfirm:
		return m.updateEndpointConfirm(msg)
	case screenDaemon:
		return m.updateDaemon(msg)
	case screenDashboard:
		return m.updateDashboard(msg)
	case screenSettings:
		return m.updateSettings(msg)
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

// --- Screen update helpers ---

func (m AppModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	newMenu, cmd := m.menu.Update(msg)
	m.menu = newMenu

	if nav, ok := asMsg[menu.NavigateTo](cmd); ok {
		switch nav.Screen {
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
	return m, cmd
}

func (m AppModel) updateEndpointList(msg tea.Msg) (tea.Model, tea.Cmd) {
	newList, cmd := m.endpointList.Update(msg)
	m.endpointList = newList

	if isMsg[endpoints.BackMsg](cmd) {
		m.current = screenMenu
		return m, nil
	}
	if nav, ok := asMsg[endpoints.ShowFormMsg](cmd); ok {
		m.endpointForm = endpoints.NewFormModel(m.endpointSvc, nav.Endpoint)
		m.current = screenEndpointForm
		return m, m.endpointForm.Init()
	}
	if nav, ok := asMsg[endpoints.ShowConfirmMsg](cmd); ok {
		m.endpointConfirm = endpoints.NewConfirmModel(m.endpointSvc, nav.Endpoint)
		m.current = screenEndpointConfirm
		return m, nil
	}

	return m, cmd
}

func (m AppModel) updateEndpointForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	newForm, cmd := m.endpointForm.Update(msg)
	m.endpointForm = newForm

	if isMsg[endpoints.BackMsg](cmd) || isMsg[endpoints.FormSubmittedMsg](cmd) {
		m.current = screenEndpoints
		m.endpointList = endpoints.NewListModel(m.endpointSvc)
		return m, nil
	}
	return m, cmd
}

func (m AppModel) updateEndpointConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	newConfirm, cmd := m.endpointConfirm.Update(msg)
	m.endpointConfirm = newConfirm

	if isMsg[endpoints.BackMsg](cmd) || isMsg[endpoints.FormSubmittedMsg](cmd) {
		m.current = screenEndpoints
		m.endpointList = endpoints.NewListModel(m.endpointSvc)
		return m, nil
	}
	return m, cmd
}

func (m AppModel) updateDaemon(msg tea.Msg) (tea.Model, tea.Cmd) {
	newDaemon, cmd := m.daemonView.Update(msg)
	m.daemonView = newDaemon

	if isMsg[daemon.BackMsg](cmd) {
		m.current = screenMenu
		return m, nil
	}
	return m, cmd
}

func (m AppModel) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	newDash, cmd := m.dashboardView.Update(msg)
	m.dashboardView = newDash

	if isMsg[dashboard.BackMsg](cmd) {
		m.current = screenMenu
		return m, nil
	}
	return m, cmd
}

func (m AppModel) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	newSettings, cmd := m.settingsView.Update(msg)
	m.settingsView = newSettings

	if isMsg[settings.BackMsg](cmd) || isMsg[settings.SavedMsg](cmd) {
		m.current = screenMenu
		return m, nil
	}
	return m, cmd
}

// --- Message helpers ---

// asMsg attempts to extract a message of type T from a tea.Cmd by executing it.
func asMsg[T any](cmd tea.Cmd) (T, bool) {
	var zero T
	if cmd == nil {
		return zero, false
	}
	msg := cmd()
	v, ok := msg.(T)
	return v, ok
}

func isMsg[T any](cmd tea.Cmd) bool {
	_, ok := asMsg[T](cmd)
	return ok
}
