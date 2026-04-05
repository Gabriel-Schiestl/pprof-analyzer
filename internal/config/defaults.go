package config

import "time"

const (
	DefaultOllamaAPIURL       = "http://localhost:11434"
	DefaultOllamaModel        = "llama3.3:70b"
	DefaultOllamaTimeoutS     = 120
	DefaultDataDir            = "~/.local/share/pprof-analyzer"
	DefaultReportsDir         = "~/pprof-reports"
	DefaultPIDFile            = "~/.local/share/pprof-analyzer/daemon.pid"
	DefaultLogFile            = "~/.local/share/pprof-analyzer/daemon.log"
	DefaultCollectIntervalS   = 300
	DefaultCPUSampleDurationS = 30
	DefaultMaxRetries         = 3
	DefaultRetryBackoffS      = 5
	DefaultCollectInterval    = 300 * time.Second
)
