# pprof-analyzer — Tasks de Desenvolvimento

> Baseado em: DESIGN.md v1.0 + SPEC.md v1.0  
> Prioridade: Fase 1 (MVP) primeiro, de dentro para fora (domínio → app → adapters → delivery)

---

## Legenda

- `[P]` — pode ser feito em paralelo com outras tasks do mesmo grupo
- `[S]` — deve ser feita em sequência (depende de task anterior)
- `[ ]` — não iniciada
- `[x]` — concluída

---

## FASE 1 — MVP

---

### BLOCO 0 — Scaffolding do Projeto

> Pré-requisito de tudo. Sequencial.

- [x] **T-001** `[S]` Inicializar módulo Go (`go mod init`) com Go 1.26 e criar estrutura de diretórios conforme DESIGN.md §2
- [x] **T-002** `[S]` Criar `Makefile` com targets: `build`, `test`, `lint`, `run`
- [x] **T-003** `[S]` Adicionar todas as dependências externas ao `go.mod` (`bubbletea`, `lipgloss`, `bubbles`, `huh`, `maroto/v2`, `ollama/api`, `google/pprof`, `oklog/run`, `google/uuid`, `stretchr/testify`)

---

### BLOCO 1 — Camada de Domínio (`internal/domain`)

> Zero dependências externas. Pode ser desenvolvido em paralelo após T-001.

- [x] **T-004** `[P]` Criar `internal/domain/endpoint.go` — structs `Endpoint`, `Credentials`, enums `Environment`, `AuthType`
- [x] **T-005** `[P]` Criar `internal/domain/profile.go` — structs `ProfileData`, `CollectionRun`, enums `ProfileType`, `RunStatus`, variável `AllProfileTypes`
- [x] **T-006** `[P]` Criar `internal/domain/analysis.go` — structs `AnalysisResult`, `ProfileFinding`, `Recommendation`, enum `Severity`
- [x] **T-007** `[P]` Criar `internal/domain/errors.go` — erros sentinela (`ErrEndpointNotFound`, `ErrDaemonAlreadyRunning`, etc.) e `RetryExhaustedError` com `Unwrap()`
- [x] **T-008** `[P]` Escrever testes unitários para `domain/errors.go` (verificar mensagens e unwrap chain)

---

### BLOCO 2 — Configuração (`internal/config`)

> Depende do BLOCO 1 (usa tipos de domínio). Tasks paralelas entre si.

- [x] **T-009** `[P]` Criar `internal/config/defaults.go` — constantes de valores padrão (URL Ollama, modelo, intervalos, diretórios)
- [x] **T-010** `[P]` Criar `internal/config/config.go` — struct `Config` com sub-structs `OllamaConfig`, `StorageConfig`, `DaemonConfig`, `CollectionConfig`; método `Load()` que lê `~/.config/pprof-analyzer/config.json` e mescla defaults

---

### BLOCO 3 — Ports da Camada de Aplicação (`internal/app/ports.go`)

> Depende do BLOCO 1. Sequencial (único arquivo, define contratos de todo o sistema).

- [x] **T-011** `[S]` Criar `internal/app/ports.go` — interfaces `ProfileCollector`, `AIProvider`, `ReportWriter`, `EndpointRepository`, `MetadataStore`, `PprofFileStore`; struct `AnalysisRequest`

---

### BLOCO 4 — Storage Adapters (`internal/storage`)

> Depende dos BLOCOs 1 e 3. Tasks paralelas entre si.

- [x] **T-012** `[P]` Criar `internal/storage/endpoint_repo.go` — implementação de `EndpointRepository` com arquivo `endpoints.json`, `sync.RWMutex` para acesso concorrente, e escrita atômica via temp+rename
- [x] **T-013** `[P]` Criar `internal/storage/metadata_store.go` — implementação de `MetadataStore` com arquivos JSON em `runs/{endpoint-id}/`, leitura do mais recente por maior timestamp no diretório
- [x] **T-014** `[P]` Criar `internal/storage/pprof_store.go` — implementação de `PprofFileStore`: salva arquivos `.pb.gz` em `pprof/{endpoint-id}/{profile-type}/` com nome `YYYYMMDD_HHMMSS.pb.gz`; `ApplyRetentionPolicy` lista e deleta excedentes mantendo apenas os 3 mais recentes
- [x] **T-015** `[P]` Criar `internal/storage/atomic.go` — helper `atomicWriteJSON(path string, v any) error` (write-to-temp + rename)
- [x] **T-016** `[P]` Escrever testes de integração para `EndpointRepository` usando `t.TempDir()` (CRUD completo + concorrência básica)
- [x] **T-017** `[P]` Escrever testes de integração para `MetadataStore` usando `t.TempDir()` (`SaveRun`, `GetLastRun`, `ListRuns`)
- [x] **T-018** `[P]` Escrever testes de integração para `PprofFileStore` usando `t.TempDir()` (save + política de retenção: verifica que só 3 arquivos permanecem após 4 saves)

---

### BLOCO 5 — Coletor (`internal/collector`)

> Depende dos BLOCOs 1 e 3. Tasks paralelas entre si.

- [x] **T-019** `[P]` Criar `internal/collector/pprof_parser.go` — função `ParseToTextSummary(rawData []byte, profileType domain.ProfileType) (string, error)`: parse binário via `google/pprof/profile`, top-30 por flat value, tabela texto `flat|flat%|cum|cum%|function`; tratamento especial para goroutine (contagem por estado + top-10 stacks); truncagem a 4.000 tokens
- [x] **T-020** `[P]` Criar `internal/collector/http_collector.go` — `HTTPCollector` implementando `ProfileCollector`: coleta concorrente com `errgroup` (max 3 goroutines), CPU profile por último e separado, retry com backoff exponencial até `maxRetries`, 404 → `ErrProfileNotAvailable` (não aborta ciclo), `RetryExhaustedError` em falha total
- [x] **T-021** `[P]` Escrever testes de integração para `HTTPCollector` usando `httptest.NewServer` (coleta bem-sucedida, retry em 500, abandono após max retries, 404 parcial)
- [x] **T-022** `[P]` Escrever testes unitários para `ParseToTextSummary` (heap, goroutine, truncagem de texto longo)

---

### BLOCO 6 — Adapter Ollama (`internal/ai/ollama`)

> Depende dos BLOCOs 1 e 3. Tasks sequenciais entre si.

- [x] **T-023** `[S]` Criar `internal/ai/ollama/client.go` — struct `OllamaClient` com `AnalyzeProfiles`: detecção de suporte a tools via `client.Show()`, Fase 1 (análise individual por perfil com system prompt específico por `ProfileType`, output JSON `ProfileFinding`), Fase 2 (análise consolidada com todos os findings, output `ConsolidatedAnalysis` + `Recommendations` + `OverallSeverity`)
- [x] **T-024** `[S]` Definir prompts e schemas JSON em `internal/ai/ollama/prompts.go` — system prompts por tipo de perfil (heap, allocs, goroutine, cpu, block, mutex, threadcreate) e prompt de análise consolidada
- [x] **T-025** `[S]` Escrever testes de integração para `OllamaClient` usando `httptest.NewServer` que simula respostas da API Ollama (análise individual, análise consolidada, fallback quando tools não suportados)

---

### BLOCO 7 — Geração de PDF (`internal/report`)

> Depende dos BLOCOs 1 e 3. Tasks sequenciais entre si.

- [x] **T-026** `[S]` Criar `internal/report/template.go` — definição da estrutura do template fixo: constantes de layout, paleta de cores de severidade (`#DC2626`, `#D97706`, `#16A34A`), helpers de formatação
- [x] **T-027** `[S]` Criar `internal/report/pdf_writer.go` — `PDFWriter` implementando `ReportWriter`: seções Cabeçalho, Sumário Executivo (badge de severidade colorido), Análise por Perfil, Análise Consolidada, Recomendações (tabela numerada + blocos de código), Rodapé; nomenclatura de arquivo `{app}_{env}_{YYYYMMDD_HHMMSS}.pdf`
- [x] **T-028** `[S]` Escrever teste golden file para `PDFWriter`: gera PDF com `AnalysisResult` fixture e verifica que arquivo é criado no caminho correto (não compara bytes, verifica existência e tamanho > 0)

---

### BLOCO 8 — Serviços de Aplicação (`internal/app`)

> Depende dos BLOCOs 1, 3, 4, 5, 6, 7. Tasks paralelas entre si.

- [x] **T-029** `[P]` Criar `internal/app/endpoint_service.go` — `EndpointService`: `Add` (valida nome/URL não vazios, gera UUID, seta timestamps), `List`, `Get`, `Update`, `Delete` delegando ao `EndpointRepository`
- [x] **T-030** `[P]` Criar `internal/app/analysis_service.go` — `AnalysisService`: orquestra ciclo completo `Collect` → `PprofFileStore.Save` → `AIProvider.AnalyzeProfiles` → `ReportWriter.Write` → `PprofFileStore.ApplyRetentionPolicy` → `MetadataStore.SaveRun`; monta `CollectionRun` com status `success/partial/failed`
- [x] **T-031** `[P]` Escrever testes unitários para `EndpointService` com mock do `EndpointRepository` (add com UUID gerado, validação de campo obrigatório, delete não encontrado)
- [x] **T-032** `[P]` Escrever testes unitários para `AnalysisService` com mocks de todas as ports (ciclo completo sucesso, falha no collector → status failed, falha no AI → run salvo sem report)

---

### BLOCO 9 — Daemon (`internal/app/daemon.go` + `internal/daemon/proc_*.go`)

> Depende do BLOCO 8. Tasks sequenciais entre si.

- [x] **T-033** `[S]` Criar `internal/daemon/proc_unix.go` (`//go:build !windows`) — `SysProcAttr` com `Setsid: true`; `StopProcess` com SIGTERM → wait 10s → SIGKILL
- [x] **T-034** `[S]` Criar `internal/daemon/proc_windows.go` (`//go:build windows`) — `SysProcAttr` com `CreationFlags: DETACHED_PROCESS`; `StopProcess` com `os.Process.Kill()`
- [x] **T-035** `[S]` Criar `internal/app/daemon.go` — `DaemonService`: `Start` (verifica PID file, re-exec com `--daemon-internal`, escreve PID file), `Stop` (lê PID, chama `StopProcess`, remove PID file), `IsRunning` (verifica PID file + `kill(pid, 0)`), `RunInProcess` (loop `oklog/run` com uma goroutine por endpoint chamando `runEndpointLoop` + goroutine de cancelamento de contexto)
- [x] **T-036** `[S]` Criar `internal/app/daemon.go` — método `runEndpointLoop`: ticker com `ep.CollectInterval`, `context.WithTimeout` com metade do intervalo por ciclo, chama `AnalysisService.RunCycle`

---

### BLOCO 10 — Estilos TUI (`internal/tui/styles`)

> Pode ser desenvolvido em paralelo com BLOCOs 8 e 9.

- [x] **T-037** `[P]` Criar `internal/tui/styles/styles.go` — definições Lip Gloss: cores primária/secundária/erro/sucesso, estilos de borda, estilos de tabela, estilos de badge de severidade (crítico/atenção/normal), larguras e paddings padrão

---

### BLOCO 11 — TUI: Menu Principal (`internal/tui/menu`)

> Depende do BLOCO 10.

- [x] **T-038** `[S]` Criar `internal/tui/menu/model.go` — `Model` Bubble Tea com lista de 4 opções (Endpoints, Daemon, Dashboard, Configurações), navegação com ↑↓, Enter seleciona, `q` sai; emite mensagem de navegação para a root

---

### BLOCO 12 — TUI: Endpoints (`internal/tui/endpoints`)

> Depende dos BLOCOs 8, 10 e 11. Tasks paralelas entre si.

- [x] **T-039** `[P]` Criar `internal/tui/endpoints/list.go` — `ListModel`: tabela com colunas Nome/URL/Ambiente/Intervalo, teclas `a` (add), `e` (edit), `d` (delete), `ESC` (volta ao menu)
- [x] **T-040** `[P]` Criar `internal/tui/endpoints/form.go` — `FormModel` com `charmbracelet/huh`: campos Nome, URL base, Ambiente (select), Intervalo (segundos), Tipo de Auth (select), Username/Password/Token (condicionais); validação inline; modo add e edit
- [x] **T-041** `[P]` Criar `internal/tui/endpoints/confirm.go` — `ConfirmModel`: tela de confirmação de remoção com nome do endpoint, teclas `y`/`n`

---

### BLOCO 13 — TUI: Daemon (`internal/tui/daemon`)

> Depende dos BLOCOs 9 e 10.

- [x] **T-042** `[S]` Criar `internal/tui/daemon/model.go` — `Model`: exibe status atual (RUNNING/STOPPED), botões Start/Stop, spinner enquanto operação está em andamento, mensagem de erro se falhar; chama `DaemonService.Start/Stop`

---

### BLOCO 14 — TUI: Dashboard (`internal/tui/dashboard`)

> Depende dos BLOCOs 8, 9 e 10.

- [x] **T-043** `[S]` Criar `internal/tui/dashboard/model.go` — `Model`: tabela com colunas Aplicação/Ambiente/Última Coleta/Alertas, status do daemon no topo, auto-refresh via `tea.Tick` a cada 5s buscando dados do `MetadataStore` e `DaemonService.IsRunning`, tecla `r` força refresh, `ESC` volta, `q` sai

---

### BLOCO 15 — TUI: Configurações (`internal/tui/settings`)

> Depende dos BLOCOs 10 e 2.

- [x] **T-044** `[S]` Criar `internal/tui/settings/model.go` — `Model` com `huh`: campos URL da API Ollama, Modelo Ollama, Diretório de relatórios; salva em `config.json` ao confirmar

---

### BLOCO 16 — TUI: Root (`internal/tui/app.go`)

> Depende de todos os BLOCOs 11–15.

- [x] **T-045** `[S]` Criar `internal/tui/app.go` — `AppModel`: gerencia `currentScreen`, composição de todos os sub-models, roteamento de mensagens de navegação entre telas, injeção de serviços (`EndpointService`, `DaemonService`, `MetadataStore`, `Config`); método `New(...)` para construção
- [ ] **T-046** `[S]` Escrever testes unitários para `AppModel.Update()` (transições de tela: menu → endpoints, menu → daemon, ESC volta ao menu)

---

### BLOCO 17 — Entry Point (`cmd/pprof-analyzer/main.go`)

> Depende de todos os BLOCOs anteriores.

- [x] **T-047** `[S]` Criar `cmd/pprof-analyzer/main.go` — wire manual de todas as dependências conforme DESIGN.md §13: `config.Load()`, instancia todos os adapters e serviços, detecta flag `--daemon-internal` para modo daemon vs modo TUI, configura `signal.NotifyContext`

---

### BLOCO 18 — Build e Distribuição

> Depende do BLOCO 17. Tasks paralelas entre si.

- [x] **T-048** `[P]` Criar `Dockerfile` — multi-stage build: stage `builder` com Go 1.26, stage final com imagem mínima (`scratch` ou `alpine`), copia binário, `ENTRYPOINT`
- [x] **T-049** `[P]` Criar `.goreleaser.yaml` — targets Linux/macOS/Windows (amd64 + arm64), geração de checksum, compressão do binário

---

### BLOCO 19 — Smoke Test E2E

> Depende do BLOCO 17.

- [x] **T-050** `[S]` Escrever script de smoke test manual (`scripts/smoke_test.sh`): sobe servidor pprof de exemplo com `net/http/pprof`, cadastra endpoint, inicia daemon, aguarda 1 ciclo, verifica que PDF foi gerado no diretório de reports

---

## FASE 2 — Análise Temporal *(futuro)*

- [ ] **T-F2-001** Adicionar campo `trend` ao `AnalysisResult` com delta heap/goroutine em relação à coleta anterior
- [ ] **T-F2-002** Adaptar `MetadataStore.ListRuns` para retornar N últimas runs com dados de perfil para comparação
- [ ] **T-F2-003** Adicionar seção "Tendências" ao template PDF com gráfico textual de evolução
- [ ] **T-F2-004** Adaptar prompt consolidado do Ollama para incluir histórico das últimas 3 runs

---

## FASE 3 — Notificações *(futuro)*

- [ ] **T-F3-001** Definir interface `Notifier` em `internal/app/ports.go`
- [ ] **T-F3-002** Implementar adapter de email (SMTP)
- [ ] **T-F3-003** Implementar adapter de Slack (webhook)
- [ ] **T-F3-004** Implementar adapter de webhook genérico
- [ ] **T-F3-005** Adicionar campos de configuração de notificação em `config.go`
- [ ] **T-F3-006** Chamar `Notifier.Notify` no `AnalysisService` após relatório gerado com severidade crítica

---

## FASE 5 — Multi-provider IA *(futuro)*

- [ ] **T-F5-001** Criar `internal/ai/langchain/client.go` implementando `AIProvider` via `langchaingo`
- [ ] **T-F5-002** Adicionar campo `provider` em `OllamaConfig` e lógica de seleção de adapter em `main.go`
- [ ] **T-F5-003** Adicionar opção de provedor na tela de Configurações da TUI

---

## Diagrama de Dependências (Fase 1)

```
BLOCO 0 (Scaffolding)
    └── BLOCO 1 (Domain)
            ├── BLOCO 2 (Config)
            └── BLOCO 3 (Ports)
                    ├── BLOCO 4 (Storage) ──────────────────┐
                    ├── BLOCO 5 (Collector) ────────────────┤
                    ├── BLOCO 6 (AI/Ollama) ────────────────┤
                    └── BLOCO 7 (Report/PDF) ───────────────┤
                                                            ▼
                                                    BLOCO 8 (App Services)
                                                            │
                                                    BLOCO 9 (Daemon)
                                                            │
                                        ┌───────────────────┘
                                        │
                    BLOCO 10 (TUI Styles)
                            ├── BLOCO 11 (Menu)
                            ├── BLOCO 12 (Endpoints TUI)
                            ├── BLOCO 13 (Daemon TUI)
                            ├── BLOCO 14 (Dashboard TUI)
                            └── BLOCO 15 (Settings TUI)
                                        │
                                BLOCO 16 (TUI Root)
                                        │
                                BLOCO 17 (main.go)
                                        │
                            ┌───────────┴───────────┐
                    BLOCO 18 (Build)        BLOCO 19 (E2E)
```
