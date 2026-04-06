package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/ai/ollama"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/app"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/collector"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/config"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/report"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/storage"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/tui"
)

const version = "0.1.0"

func main() {
	daemonInternal := flag.Bool("daemon-internal", false, "run as background daemon (internal use)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	setupLogging(cfg, *daemonInternal)

	if *daemonInternal {
		runDaemon(cfg)
		return
	}

	runTUI(cfg)
}

func runDaemon(cfg *config.Config) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	svc, err := buildDaemonService(cfg)
	if err != nil {
		slog.Error("failed to build daemon service", "err", err)
		os.Exit(1)
	}

	slog.Info("daemon starting", "version", version)
	if err := svc.RunInProcess(ctx); err != nil && err != context.Canceled {
		slog.Error("daemon exited with error", "err", err)
		os.Exit(1)
	}
}

func runTUI(cfg *config.Config) {
	dataDir := cfg.ExpandedDataDir()
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "error creating data dir: %v\n", err)
		os.Exit(1)
	}

	endpointRepo := storage.NewEndpointRepository(dataDir)
	metadataStore := storage.NewMetadataStore(dataDir)
	pprofStore := storage.NewPprofFileStore(dataDir)

	ollamaClient, err := ollama.NewOllamaClient(cfg.Ollama.APIURL, cfg.Ollama.Model, cfg.OllamaTimeout())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating ollama client: %v\n", err)
		os.Exit(1)
	}

	pdfWriter := report.NewPDFWriter(cfg.ExpandedReportsDir())
	httpCollector := collector.NewHTTPCollector(
		cfg.OllamaTimeout(),
		cfg.Collection.MaxRetries,
		cfg.RetryBackoff(),
		cfg.CPUSampleDuration(),
	)

	analysisSvc := app.NewAnalysisService(httpCollector, ollamaClient, pdfWriter, pprofStore, metadataStore, version)
	endpointSvc := app.NewEndpointService(endpointRepo)
	daemonSvc := app.NewDaemonService(cfg.ExpandedPIDFile(), cfg.ExpandedLogFile(), endpointRepo, analysisSvc)

	model := tui.New(endpointSvc, daemonSvc, endpointRepo, metadataStore, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func buildDaemonService(cfg *config.Config) (*app.DaemonService, error) {
	dataDir := cfg.ExpandedDataDir()
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}

	endpointRepo := storage.NewEndpointRepository(dataDir)
	metadataStore := storage.NewMetadataStore(dataDir)
	pprofStore := storage.NewPprofFileStore(dataDir)

	ollamaClient, err := ollama.NewOllamaClient(cfg.Ollama.APIURL, cfg.Ollama.Model, cfg.OllamaTimeout())
	if err != nil {
		return nil, err
	}

	pdfWriter := report.NewPDFWriter(cfg.ExpandedReportsDir())
	httpCollector := collector.NewHTTPCollector(
		cfg.OllamaTimeout(),
		cfg.Collection.MaxRetries,
		cfg.RetryBackoff(),
		cfg.CPUSampleDuration(),
	)

	analysisSvc := app.NewAnalysisService(httpCollector, ollamaClient, pdfWriter, pprofStore, metadataStore, version)
	return app.NewDaemonService(cfg.ExpandedPIDFile(), cfg.ExpandedLogFile(), endpointRepo, analysisSvc), nil
}

func setupLogging(cfg *config.Config, isDaemon bool) {
	level := slog.LevelInfo
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if isDaemon {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
	_ = cfg // cfg reserved for future log level config
}
