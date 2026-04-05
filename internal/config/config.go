package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OllamaConfig holds settings for the Ollama AI provider.
type OllamaConfig struct {
	APIURL    string `json:"api_url"`
	Model     string `json:"model"`
	TimeoutS  int    `json:"timeout_s"`
}

// StorageConfig holds filesystem paths for data and reports.
type StorageConfig struct {
	DataDir    string `json:"data_dir"`
	ReportsDir string `json:"reports_dir"`
}

// DaemonConfig holds paths for daemon state files.
type DaemonConfig struct {
	PIDFile string `json:"pid_file"`
	LogFile string `json:"log_file"`
}

// CollectionConfig holds parameters governing the collection cycle.
type CollectionConfig struct {
	DefaultIntervalS    int `json:"default_interval_s"`
	CPUSampleDurationS  int `json:"cpu_sample_duration_s"`
	MaxRetries          int `json:"max_retries"`
	RetryBackoffS       int `json:"retry_backoff_s"`
}

// Config is the root configuration struct for pprof-analyzer.
type Config struct {
	Ollama     OllamaConfig     `json:"ollama"`
	Storage    StorageConfig    `json:"storage"`
	Daemon     DaemonConfig     `json:"daemon"`
	Collection CollectionConfig `json:"collection"`
}

// Load reads config from ~/.config/pprof-analyzer/config.json and merges
// any missing fields with built-in defaults.
func Load() (*Config, error) {
	cfg := defaults()

	path, err := configPath()
	if err != nil {
		return cfg, nil // best-effort: return defaults if path resolution fails
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil // first run — no file yet
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	applyDefaults(cfg)
	return cfg, nil
}

// Save persists the config to disk, creating directories as needed.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// OllamaTimeout returns the Ollama timeout as a time.Duration.
func (c *Config) OllamaTimeout() time.Duration {
	return time.Duration(c.Ollama.TimeoutS) * time.Second
}

// CPUSampleDuration returns the CPU profile sampling duration.
func (c *Config) CPUSampleDuration() time.Duration {
	return time.Duration(c.Collection.CPUSampleDurationS) * time.Second
}

// RetryBackoff returns the retry backoff duration.
func (c *Config) RetryBackoff() time.Duration {
	return time.Duration(c.Collection.RetryBackoffS) * time.Second
}

// ExpandedDataDir returns DataDir with ~ expanded.
func (c *Config) ExpandedDataDir() string {
	return expandHome(c.Storage.DataDir)
}

// ExpandedReportsDir returns ReportsDir with ~ expanded.
func (c *Config) ExpandedReportsDir() string {
	return expandHome(c.Storage.ReportsDir)
}

// ExpandedPIDFile returns PIDFile with ~ expanded.
func (c *Config) ExpandedPIDFile() string {
	return expandHome(c.Daemon.PIDFile)
}

// ExpandedLogFile returns LogFile with ~ expanded.
func (c *Config) ExpandedLogFile() string {
	return expandHome(c.Daemon.LogFile)
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pprof-analyzer", "config.json"), nil
}

func defaults() *Config {
	return &Config{
		Ollama: OllamaConfig{
			APIURL:   DefaultOllamaAPIURL,
			Model:    DefaultOllamaModel,
			TimeoutS: DefaultOllamaTimeoutS,
		},
		Storage: StorageConfig{
			DataDir:    DefaultDataDir,
			ReportsDir: DefaultReportsDir,
		},
		Daemon: DaemonConfig{
			PIDFile: DefaultPIDFile,
			LogFile: DefaultLogFile,
		},
		Collection: CollectionConfig{
			DefaultIntervalS:   DefaultCollectIntervalS,
			CPUSampleDurationS: DefaultCPUSampleDurationS,
			MaxRetries:         DefaultMaxRetries,
			RetryBackoffS:      DefaultRetryBackoffS,
		},
	}
}

func applyDefaults(cfg *Config) {
	if cfg.Ollama.APIURL == "" {
		cfg.Ollama.APIURL = DefaultOllamaAPIURL
	}
	if cfg.Ollama.Model == "" {
		cfg.Ollama.Model = DefaultOllamaModel
	}
	if cfg.Ollama.TimeoutS == 0 {
		cfg.Ollama.TimeoutS = DefaultOllamaTimeoutS
	}
	if cfg.Storage.DataDir == "" {
		cfg.Storage.DataDir = DefaultDataDir
	}
	if cfg.Storage.ReportsDir == "" {
		cfg.Storage.ReportsDir = DefaultReportsDir
	}
	if cfg.Daemon.PIDFile == "" {
		cfg.Daemon.PIDFile = DefaultPIDFile
	}
	if cfg.Daemon.LogFile == "" {
		cfg.Daemon.LogFile = DefaultLogFile
	}
	if cfg.Collection.DefaultIntervalS == 0 {
		cfg.Collection.DefaultIntervalS = DefaultCollectIntervalS
	}
	if cfg.Collection.CPUSampleDurationS == 0 {
		cfg.Collection.CPUSampleDurationS = DefaultCPUSampleDurationS
	}
	if cfg.Collection.MaxRetries == 0 {
		cfg.Collection.MaxRetries = DefaultMaxRetries
	}
	if cfg.Collection.RetryBackoffS == 0 {
		cfg.Collection.RetryBackoffS = DefaultRetryBackoffS
	}
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
