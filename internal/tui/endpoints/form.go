package endpoints

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/app"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui/styles"
)

// FormSubmittedMsg is sent when the form is successfully submitted.
type FormSubmittedMsg struct{}

// formData holds form field values on the heap so that huh's Value() pointers
// remain stable across Bubble Tea model copies.
type formData struct {
	name        string
	baseURL     string
	environment string
	intervalS   string
	authType    string
	username    string
	password    string
	token       string
}

// FormModel is the add/edit endpoint form model.
type FormModel struct {
	form     *huh.Form
	svc      *app.EndpointService
	editing  *domain.Endpoint
	errorMsg string
	data     *formData
}

// NewFormModel creates a form for adding or editing an endpoint.
// Pass nil for ep to create a new endpoint.
func NewFormModel(svc *app.EndpointService, ep *domain.Endpoint) FormModel {
	data := &formData{}

	if ep != nil {
		data.name = ep.Name
		data.baseURL = ep.BaseURL
		data.environment = string(ep.Environment)
		data.intervalS = strconv.Itoa(int(ep.CollectInterval.Seconds()))
		data.authType = string(ep.Credentials.AuthType)
		data.username = ep.Credentials.Username
		data.password = ep.Credentials.Password
		data.token = ep.Credentials.Token
	} else {
		data.environment = string(domain.EnvDevelopment)
		data.authType = string(domain.AuthNone)
		data.intervalS = "300"
	}

	m := FormModel{svc: svc, editing: ep, data: data}
	m.form = buildForm(data)
	return m
}

func buildForm(data *formData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("Identifier for the application").
				Value(&data.name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Base URL").
				Description("e.g. http://localhost:6060").
				Value(&data.baseURL),

			huh.NewSelect[string]().
				Title("Environment").
				Options(
					huh.NewOption("development", string(domain.EnvDevelopment)),
					huh.NewOption("staging", string(domain.EnvStaging)),
					huh.NewOption("production", string(domain.EnvProduction)),
				).
				Value(&data.environment),

			huh.NewInput().
				Title("Collect interval (seconds)").
				Description("Default: 300").
				Value(&data.intervalS),

			huh.NewSelect[string]().
				Title("Authentication").
				Options(
					huh.NewOption("None", string(domain.AuthNone)),
					huh.NewOption("Basic Auth", string(domain.AuthBasic)),
					huh.NewOption("Bearer Token", string(domain.AuthBearerToken)),
				).
				Value(&data.authType),
		),
		huh.NewGroup(
			huh.NewInput().Title("Username").Value(&data.username),
			huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&data.password),
		).WithHideFunc(func() bool { return data.authType != string(domain.AuthBasic) }),

		huh.NewGroup(
			huh.NewInput().Title("Bearer Token").EchoMode(huh.EchoModePassword).Value(&data.token),
		).WithHideFunc(func() bool { return data.authType != string(domain.AuthBearerToken) }),
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
			m.form = buildForm(m.data)
			return m, m.form.Init()
		}
		return m, func() tea.Msg { return FormSubmittedMsg{} }
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
	intervalS, _ := strconv.Atoi(m.data.intervalS)
	if intervalS <= 0 {
		intervalS = 300
	}

	ep := domain.Endpoint{
		Name:            m.data.name,
		BaseURL:         m.data.baseURL,
		Environment:     domain.Environment(m.data.environment),
		CollectInterval: time.Duration(intervalS) * time.Second,
		Credentials: domain.Credentials{
			AuthType: domain.AuthType(m.data.authType),
			Username: m.data.username,
			Password: m.data.password,
			Token:    m.data.token,
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
