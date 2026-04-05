package endpoints

import (
	"strconv"
	"time"

	"github.com/charmbracelet/huh"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gabri/pprof-analyzer/internal/app"
	"github.com/gabri/pprof-analyzer/internal/domain"
	"github.com/gabri/pprof-analyzer/internal/tui/styles"
)

// FormSubmittedMsg is sent when the form is successfully submitted.
type FormSubmittedMsg struct{}

// FormModel is the add/edit endpoint form model.
type FormModel struct {
	form     *huh.Form
	svc      *app.EndpointService
	editing  *domain.Endpoint
	errorMsg string

	// form fields
	name        string
	baseURL     string
	environment string
	intervalS   string
	authType    string
	username    string
	password    string
	token       string
}

// NewFormModel creates a form for adding or editing an endpoint.
// Pass nil for ep to create a new endpoint.
func NewFormModel(svc *app.EndpointService, ep *domain.Endpoint) FormModel {
	m := FormModel{svc: svc, editing: ep}

	if ep != nil {
		m.name = ep.Name
		m.baseURL = ep.BaseURL
		m.environment = string(ep.Environment)
		m.intervalS = strconv.Itoa(int(ep.CollectInterval.Seconds()))
		m.authType = string(ep.Credentials.AuthType)
		m.username = ep.Credentials.Username
		m.password = ep.Credentials.Password
		m.token = ep.Credentials.Token
	} else {
		m.environment = string(domain.EnvDevelopment)
		m.authType = string(domain.AuthNone)
		m.intervalS = "300"
	}

	m.form = buildForm(&m)
	return m
}

func buildForm(m *FormModel) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("Identifier for the application").
				Value(&m.name).
				Validate(func(s string) error {
					if s == "" {
						return huh.ErrUserAborted
					}
					return nil
				}),

			huh.NewInput().
				Title("Base URL").
				Description("e.g. http://localhost:6060").
				Value(&m.baseURL),

			huh.NewSelect[string]().
				Title("Environment").
				Options(
					huh.NewOption("development", string(domain.EnvDevelopment)),
					huh.NewOption("staging", string(domain.EnvStaging)),
					huh.NewOption("production", string(domain.EnvProduction)),
				).
				Value(&m.environment),

			huh.NewInput().
				Title("Collect interval (seconds)").
				Description("Default: 300").
				Value(&m.intervalS),

			huh.NewSelect[string]().
				Title("Authentication").
				Options(
					huh.NewOption("None", string(domain.AuthNone)),
					huh.NewOption("Basic Auth", string(domain.AuthBasic)),
					huh.NewOption("Bearer Token", string(domain.AuthBearerToken)),
				).
				Value(&m.authType),
		),
		huh.NewGroup(
			huh.NewInput().Title("Username").Value(&m.username),
			huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&m.password),
		).WithHideFunc(func() bool { return m.authType != string(domain.AuthBasic) }),

		huh.NewGroup(
			huh.NewInput().Title("Bearer Token").EchoMode(huh.EchoModePassword).Value(&m.token),
		).WithHideFunc(func() bool { return m.authType != string(domain.AuthBearerToken) }),
	)
}

// Init implements tea.Model.
func (m FormModel) Init() tea.Cmd { return m.form.Init() }

// Update implements tea.Model.
func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg { return BackMsg{} }
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		if err := m.submit(); err != nil {
			m.errorMsg = err.Error()
		} else {
			return m, func() tea.Msg { return FormSubmittedMsg{} }
		}
	}
	if m.form.State == huh.StateAborted {
		return m, func() tea.Msg { return BackMsg{} }
	}

	return m, cmd
}

// View implements tea.Model.
func (m FormModel) View() string {
	title := "Add Endpoint"
	if m.editing != nil {
		title = "Edit Endpoint: " + m.editing.Name
	}

	out := styles.SectionStyle.Render(title) + "\n" + m.form.View()
	if m.errorMsg != "" {
		out += "\n" + styles.ErrorStyle.Render(m.errorMsg)
	}
	return styles.AppStyle.Render(out)
}

func (m *FormModel) submit() error {
	intervalS, _ := strconv.Atoi(m.intervalS)
	if intervalS <= 0 {
		intervalS = 300
	}

	ep := domain.Endpoint{
		Name:            m.name,
		BaseURL:         m.baseURL,
		Environment:     domain.Environment(m.environment),
		CollectInterval: time.Duration(intervalS) * time.Second,
		Credentials: domain.Credentials{
			AuthType: domain.AuthType(m.authType),
			Username: m.username,
			Password: m.password,
			Token:    m.token,
		},
	}

	if m.editing != nil {
		ep.ID = m.editing.ID
		ep.CreatedAt = m.editing.CreatedAt
		return m.svc.Update(ep)
	}

	_, err := m.svc.Add(ep)
	return err
}
