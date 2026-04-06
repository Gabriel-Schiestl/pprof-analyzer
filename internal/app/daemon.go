package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/run"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/daemon"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
)

// DaemonService manages the background collection daemon process.
type DaemonService struct {
	pidFile         string
	logFile         string
	endpointRepo    EndpointRepository
	analysisService *AnalysisService
}

// NewDaemonService creates a DaemonService with all required dependencies.
func NewDaemonService(
	pidFile, logFile string,
	endpointRepo EndpointRepository,
	analysisService *AnalysisService,
) *DaemonService {
	return &DaemonService{
		pidFile:         pidFile,
		logFile:         logFile,
		endpointRepo:    endpointRepo,
		analysisService: analysisService,
	}
}

// Start spawns the daemon as a detached background process.
// It re-execs the current binary with --daemon-internal flag.
func (d *DaemonService) Start() error {
	if d.IsRunning() {
		return domain.ErrDaemonAlreadyRunning
	}

	logFile, err := os.OpenFile(d.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open daemon log file: %w", err)
	}

	cmd := exec.Command(os.Args[0], "--daemon-internal")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = daemon.SysProcAttr()

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start daemon process: %w", err)
	}

	logFile.Close()

	if err := os.WriteFile(d.pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}

	slog.Info("daemon started", "pid", cmd.Process.Pid)
	return nil
}

// Stop sends a stop signal to the daemon and removes the PID file.
func (d *DaemonService) Stop() error {
	pid, err := d.readPID()
	if err != nil {
		return domain.ErrDaemonNotRunning
	}

	if err := daemon.StopProcess(pid); err != nil {
		return fmt.Errorf("stop daemon process: %w", err)
	}

	os.Remove(d.pidFile)
	slog.Info("daemon stopped", "pid", pid)
	return nil
}

// IsRunning returns true if a daemon process is currently active.
func (d *DaemonService) IsRunning() bool {
	pid, err := d.readPID()
	if err != nil {
		return false
	}
	return daemon.IsProcessRunning(pid)
}

// RunInProcess is called when the binary is started with --daemon-internal.
// It runs the collection loop in the current process until ctx is cancelled.
func (d *DaemonService) RunInProcess(ctx context.Context) error {
	endpoints, err := d.endpointRepo.List()
	if err != nil {
		return fmt.Errorf("load endpoints: %w", err)
	}

	if len(endpoints) == 0 {
		slog.Warn("no endpoints configured; daemon exiting")
		return nil
	}

	var g run.Group

	for _, ep := range endpoints {
		epCtx, cancel := context.WithCancel(ctx)
		g.Add(
			func() error { return d.runEndpointLoop(epCtx, ep) },
			func(_ error) { cancel() },
		)
	}

	// Shutdown goroutine — exits when ctx is done
	g.Add(
		func() error {
			<-ctx.Done()
			return ctx.Err()
		},
		func(_ error) {},
	)

	return g.Run()
}

// runEndpointLoop ticks on ep.CollectInterval and runs a collection cycle per tick.
func (d *DaemonService) runEndpointLoop(ctx context.Context, ep domain.Endpoint) error {
	interval := ep.CollectInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("endpoint loop started", "endpoint", ep.Name, "interval", interval)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			cycleTimeout := interval / 2
			if cycleTimeout < 30*time.Second {
				cycleTimeout = 30 * time.Second
			}
			cycleCtx, cancel := context.WithTimeout(ctx, cycleTimeout)
			if err := d.analysisService.RunCycle(cycleCtx, ep); err != nil {
				slog.Error("collection cycle error", "endpoint", ep.Name, "err", err)
			}
			cancel()
		}
	}
}

func (d *DaemonService) readPID() (int, error) {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file contents: %w", err)
	}
	return pid, nil
}
