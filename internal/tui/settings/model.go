package settings

import (
	"github.com/charmbracelet/huh"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gabri/pprof-analyzer/internal/config"
	"github.com/gabri/pprof-analyzer/internal/tui/styles"
)

// BackMsg navigates back to the main menu.
type BackMsg struct{}

// SavedMsg is emitted when settings are saved.
type SavedMsg struct{}

// Model is the Bubble Tea model for the settings screen.
type Model struct {
	form       *huh.Form
	cfg        *config.Config
	errorMsg   string
	apiURL     string
	model      string
	reportsDir string
}

// New creates a settings model pre-filled from the current config.
func New(cfg *config.Config) Model {
	m := Model{
		cfg:        cfg,
		apiURL:     cfg.Ollama.APIURL,
		model:      cfg.Ollama.Model,
		reportsDir: cfg.Storage.ReportsDir,
	}
	m.form = buildForm(&m)
	return m
}

func buildForm(m *Model) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Ollama API URL").
				Description("e.g. http://localhost:11434").
				Value(&m.apiURL),

			huh.NewInput().
				Title("Ollama Model").
				Description("e.g. llama3.3:70b or qwen2.5-coder:32b").
				Value(&m.model),

			huh.NewInput().
				Title("Reports Directory").
				Description("Where PDF reports will be saved").
				Value(&m.reportsDir),
		),
	)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return m.form.Init() }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
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
		if err := m.save(); err != nil {
			m.errorMsg = err.Error()
		} else {
			return m, func() tea.Msg { return SavedMsg{} }
		}
	}
	if m.form.State == huh.StateAborted {
		return m, func() tea.Msg { return BackMsg{} }
	}

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	out := styles.SectionStyle.Render("Settings") + "\n" + m.form.View()
	if m.errorMsg != "" {
		out += "\n" + styles.ErrorStyle.Render(m.errorMsg)
	}
	return styles.AppStyle.Render(out)
}

func (m *Model) save() error {
	m.cfg.Ollama.APIURL = m.apiURL
	m.cfg.Ollama.Model = m.model
	m.cfg.Storage.ReportsDir = m.reportsDir
	return m.cfg.Save()
}
