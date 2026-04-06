package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/config"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui/styles"
)

// BackMsg navigates back to the main menu.
type BackMsg struct{}

// SavedMsg is emitted when settings are saved.
type SavedMsg struct{}

// settingsData holds form field values on the heap so that huh's Value() pointers
// remain stable across Bubble Tea model copies.
type settingsData struct {
	apiURL     string
	model      string
	reportsDir string
}

// Model is the Bubble Tea model for the settings screen.
type Model struct {
	form     *huh.Form
	cfg      *config.Config
	errorMsg string
	data     *settingsData
}

// New creates a settings model pre-filled from the current config.
func New(cfg *config.Config) Model {
	data := &settingsData{
		apiURL:     cfg.Ollama.APIURL,
		model:      cfg.Ollama.Model,
		reportsDir: cfg.Storage.ReportsDir,
	}
	m := Model{cfg: cfg, data: data}
	m.form = buildForm(data)
	return m
}

func buildForm(data *settingsData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Ollama API URL").
				Description("e.g. http://localhost:11434").
				Value(&data.apiURL),

			huh.NewInput().
				Title("Ollama Model").
				Description("e.g. llama3.3:70b or qwen2.5-coder:32b").
				Value(&data.model),

			huh.NewInput().
				Title("Reports Directory").
				Description("Where PDF reports will be saved").
				Value(&data.reportsDir),
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
	m.cfg.Ollama.APIURL = m.data.apiURL
	m.cfg.Ollama.Model = m.data.model
	m.cfg.Storage.ReportsDir = m.data.reportsDir
	return m.cfg.Save()
}
