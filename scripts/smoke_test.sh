#!/usr/bin/env bash
# Smoke test: spin up a pprof server, run one collection cycle, verify PDF output.
set -euo pipefail

BINARY="${1:-./build/pprof-analyzer}"
REPORTS_DIR="$(mktemp -d)"
DATA_DIR="$(mktemp -d)"
CONFIG_FILE="$(mktemp)"

cleanup() {
  kill "$PPROF_PID" 2>/dev/null || true
  kill "$DAEMON_PID" 2>/dev/null || true
  rm -rf "$REPORTS_DIR" "$DATA_DIR" "$CONFIG_FILE"
}
trap cleanup EXIT

echo "==> Building smoke test server..."
cat > /tmp/smoke_server.go << 'EOF'
package main

import (
  "log"
  "net/http"
  _ "net/http/pprof"
)

func main() {
  log.Println("pprof server listening on :16060")
  log.Fatal(http.ListenAndServe(":16060", nil))
}
EOF

go run /tmp/smoke_server.go &
PPROF_PID=$!
sleep 1

echo "==> Writing config..."
cat > "$CONFIG_FILE" << EOF
{
  "ollama": {
    "api_url": "http://localhost:11434",
    "model": "llama3.2:3b",
    "timeout_s": 30
  },
  "storage": {
    "data_dir": "$DATA_DIR",
    "reports_dir": "$REPORTS_DIR"
  },
  "daemon": {
    "pid_file": "$DATA_DIR/daemon.pid",
    "log_file": "$DATA_DIR/daemon.log"
  },
  "collection": {
    "default_interval_s": 10,
    "cpu_sample_duration_s": 5,
    "max_retries": 2,
    "retry_backoff_s": 1
  }
}
EOF
export XDG_CONFIG_HOME="$(dirname "$CONFIG_FILE")"

echo "==> Registering endpoint..."
# For smoke test we use the API directly via a small Go program
cat > /tmp/smoke_register.go << 'EOF'
package main

import (
  "encoding/json"
  "fmt"
  "os"
  "time"
)

type Endpoint struct {
  ID              string        `json:"id"`
  Name            string        `json:"name"`
  BaseURL         string        `json:"base_url"`
  Environment     string        `json:"environment"`
  CollectInterval time.Duration `json:"collect_interval_s"`
}

func main() {
  ep := []Endpoint{{
    ID: "smoke-test-1",
    Name: "smoke-app",
    BaseURL: "http://localhost:16060",
    Environment: "development",
    CollectInterval: 10 * time.Second,
  }}
  data, _ := json.MarshalIndent(ep, "", "  ")
  dir := os.Args[1]
  os.WriteFile(dir+"/endpoints.json", data, 0600)
  fmt.Println("Registered endpoint")
}
EOF
go run /tmp/smoke_register.go "$DATA_DIR"

echo "==> Starting daemon..."
"$BINARY" --daemon-internal &
DAEMON_PID=$!

echo "==> Waiting 20s for at least one collection cycle..."
sleep 20

echo "==> Verifying PDF output..."
PDF_COUNT=$(find "$REPORTS_DIR" -name "*.pdf" | wc -l)
if [ "$PDF_COUNT" -gt 0 ]; then
  echo "✓ Smoke test passed: $PDF_COUNT PDF(s) generated"
  find "$REPORTS_DIR" -name "*.pdf"
else
  echo "✗ Smoke test failed: no PDFs generated"
  echo "Daemon log:"
  cat "$DATA_DIR/daemon.log" 2>/dev/null || echo "(no log)"
  exit 1
fi
