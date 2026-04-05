# pprof-analyzer — Estado do Desenvolvimento

> Última atualização: 2026-04-04

---

## Resumo Executivo

O MVP (Fase 1) foi **completamente implementado**. O projeto está buildando sem erros, os testes unitários e de integração passam em todas as camadas críticas.

---

## O que foi desenvolvido

### BLOCO 0 — Scaffolding ✅
- Módulo Go inicializado (`go mod init github.com/gabri/pprof-analyzer`)
- Estrutura de diretórios criada conforme DESIGN.md §2
- `Makefile` com targets `build`, `test`, `lint`, `run`, `clean`, `tidy`
- Todas as dependências adicionadas: `bubbletea`, `lipgloss`, `bubbles`, `huh`, `maroto/v2`, `ollama/api`, `google/pprof`, `oklog/run`, `google/uuid`, `stretchr/testify`

### BLOCO 1 — Camada de Domínio ✅
- [internal/domain/endpoint.go](internal/domain/endpoint.go) — `Endpoint`, `Credentials`, `Environment`, `AuthType`
- [internal/domain/profile.go](internal/domain/profile.go) — `ProfileData`, `CollectionRun`, `ProfileType`, `RunStatus`, `AllProfileTypes`
- [internal/domain/analysis.go](internal/domain/analysis.go) — `AnalysisResult`, `ProfileFinding`, `Recommendation`, `Severity`
- [internal/domain/errors.go](internal/domain/errors.go) — erros sentinela + `RetryExhaustedError` com `Unwrap()`
- Testes: 3 testes passando (sentinels, mensagem, unwrap chain)

### BLOCO 2 — Configuração ✅
- [internal/config/defaults.go](internal/config/defaults.go) — constantes de valores padrão
- [internal/config/config.go](internal/config/config.go) — `Config` com `Load()`, `Save()`, merge de defaults, expansão de `~`

### BLOCO 3 — Ports da Aplicação ✅
- [internal/app/ports.go](internal/app/ports.go) — interfaces `ProfileCollector`, `AIProvider`, `ReportWriter`, `EndpointRepository`, `MetadataStore`, `PprofFileStore`; struct `AnalysisRequest`

### BLOCO 4 — Storage Adapters ✅
- [internal/storage/atomic.go](internal/storage/atomic.go) — escrita atômica via temp+rename
- [internal/storage/endpoint_repo.go](internal/storage/endpoint_repo.go) — `EndpointRepository` com `sync.RWMutex` + JSON persistido
- [internal/storage/metadata_store.go](internal/storage/metadata_store.go) — `MetadataStore` com arquivos JSON por run
- [internal/storage/pprof_store.go](internal/storage/pprof_store.go) — `PprofFileStore` com política de retenção (mantém 3 mais recentes)
- Testes: 9 testes passando (CRUD, concorrência, retenção)

### BLOCO 5 — Coletor ✅
- [internal/collector/pprof_parser.go](internal/collector/pprof_parser.go) — `ParseToTextSummary`: top-30 functions, tratamento especial goroutine, truncagem a 4.000 tokens
- [internal/collector/http_collector.go](internal/collector/http_collector.go) — `HTTPCollector`: coleta concorrente (max 3), CPU por último, retry com backoff exponencial, 404 não aborta ciclo
- Testes: 5 testes passando (sucesso, retry em 500, 404 parcial, abandono, cancelamento de contexto)

### BLOCO 6 — Adapter Ollama ✅
- [internal/ai/ollama/prompts.go](internal/ai/ollama/prompts.go) — system prompts por tipo de perfil + schemas JSON
- [internal/ai/ollama/client.go](internal/ai/ollama/client.go) — `OllamaClient`: análise individual por perfil (Fase 1) + análise consolidada (Fase 2), parsing de JSON com strip de markdown fences, fallback em falha
- Testes: 2 testes passando (análise completa via mock, provider indisponível)

### BLOCO 7 — Geração de PDF ✅
- [internal/report/template.go](internal/report/template.go) — constantes de layout, paleta de cores de severidade, helpers
- [internal/report/pdf_writer.go](internal/report/pdf_writer.go) — `PDFWriter` com seções: Cabeçalho, Sumário Executivo, Análise por Perfil, Análise Consolidada, Recomendações, Rodapé; nomenclatura `{app}/{env}/{YYYY-MM-DD}/{app}_{env}_{ts}.pdf`
- Testes: 2 testes passando (geração do arquivo, verificação de path)

### BLOCO 8 — Serviços de Aplicação ✅
- [internal/app/endpoint_service.go](internal/app/endpoint_service.go) — `EndpointService`: CRUD com validação, UUID gerado, timestamps
- [internal/app/analysis_service.go](internal/app/analysis_service.go) — `AnalysisService`: orquestra ciclo completo com defer para salvar run mesmo em falha
- Testes: 7 testes passando (add, validação, delete, ciclo completo, falha collector, falha AI)

### BLOCO 9 — Daemon ✅
- [internal/daemon/proc_unix.go](internal/daemon/proc_unix.go) — `SysProcAttr` com `Setsid`, `StopProcess` com SIGTERM→SIGKILL
- [internal/daemon/proc_windows.go](internal/daemon/proc_windows.go) — `DETACHED_PROCESS`, `StopProcess` com Kill
- [internal/app/daemon.go](internal/app/daemon.go) — `DaemonService`: Start (re-exec + PID file), Stop, IsRunning, `RunInProcess` via `oklog/run`, `runEndpointLoop` com ticker

### BLOCO 10-16 — TUI ✅
- [internal/tui/styles/styles.go](internal/tui/styles/styles.go) — paleta Lip Gloss completa: cores, badges, tabelas, status
- [internal/tui/menu/model.go](internal/tui/menu/model.go) — menu principal com navegação ↑↓ + Enter
- [internal/tui/endpoints/list.go](internal/tui/endpoints/list.go) — tabela de endpoints com atalhos a/e/d
- [internal/tui/endpoints/form.go](internal/tui/endpoints/form.go) — formulário add/edit com `huh` (campos condicionais por auth type)
- [internal/tui/endpoints/confirm.go](internal/tui/endpoints/confirm.go) — confirmação de remoção y/n
- [internal/tui/daemon/model.go](internal/tui/daemon/model.go) — controle start/stop com spinner
- [internal/tui/dashboard/model.go](internal/tui/dashboard/model.go) — tabela de status com auto-refresh a cada 5s
- [internal/tui/settings/model.go](internal/tui/settings/model.go) — formulário de configurações salvo em config.json
- [internal/tui/app.go](internal/tui/app.go) — `AppModel`: roteamento entre telas via message passing

### BLOCO 17 — Entry Point ✅
- [cmd/pprof-analyzer/main.go](cmd/pprof-analyzer/main.go) — wire manual de todas as dependências, detecção de `--daemon-internal`, logging estruturado com `slog`

### BLOCO 18 — Build e Distribuição ✅
- [Dockerfile](Dockerfile) — multi-stage: `golang:1.26-alpine` → `alpine:3.21`
- [.goreleaser.yaml](.goreleaser.yaml) — Linux/macOS/Windows × amd64/arm64 com checksums

### BLOCO 19 — Smoke Test ✅
- [scripts/smoke_test.sh](scripts/smoke_test.sh) — servidor pprof de exemplo + daemon + verificação de PDF

---

## Resultado dos Testes

```
PASS  github.com/gabri/pprof-analyzer/internal/domain       (3 testes)
PASS  github.com/gabri/pprof-analyzer/internal/storage      (9 testes)
PASS  github.com/gabri/pprof-analyzer/internal/app          (7 testes)
PASS  github.com/gabri/pprof-analyzer/internal/report       (2 testes)
PASS  github.com/gabri/pprof-analyzer/internal/collector    (5 testes)
```

`go build ./...` — **sem erros**

---

## Próxima Task

**T-046** — Escrever testes unitários para `AppModel.Update()` (transições de tela: menu → endpoints, menu → daemon, ESC volta ao menu) em `internal/tui/app_test.go`.

Esta é a única task da Fase 1 ainda pendente. Após concluí-la, o MVP estará 100% completo.

---

## Fase 2 (próxima etapa — futuro)

- **T-F2-001** a **T-F2-004**: Análise temporal — tendências entre coletas, relatório comparativo, histórico nos prompts Ollama
