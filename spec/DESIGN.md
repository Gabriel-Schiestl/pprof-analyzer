# pprof-analyzer — Design Técnico

> Versão 1.0 — Abril 2026  
> Baseado em: SPEC.md v1.0

---

## 0. Premissas Assumidas pelo Design

As seguintes decisões de contexto foram tomadas para guiar este documento. Devem ser confirmadas ou ajustadas antes de iniciar o desenvolvimento:

| Premissa | Valor Confirmado | Justificativa |
|----------|----------------|---------------|
| Sistemas operacionais alvo | Linux, macOS, Windows | Ferramenta open source; maior adoção |
| Persistência do daemon | Processo em background via PID file (sem integração com systemd/launchd/Windows Service no MVP) | Minimiza dependências do SO; suficiente para o MVP |
| Formato de configuração | JSON | Escolha do usuário; zero dependências (encoding/json stdlib) |
| Armazenamento de metadados | Sistema de arquivos (arquivos JSON por run) | Escolha do usuário; sem dependência de banco de dados no MVP |
| Armazenamento de arquivos pprof | Sistema de arquivos (pastas organizadas por app/perfil/timestamp) | Sem overhead de encoding; compatível com ferramentas pprof nativas |
| Versão do Go | 1.26 (mais recente estável) | Escolha do usuário; todas as features modernas disponíveis |
| Cobertura de testes | Apenas partes críticas (collector, retention policy, AI pipeline, report) | Escolha do usuário; pragmático para MVP |

---

## 1. Visão Arquitetural

### 1.1 Estilo: Hexagonal (Ports & Adapters)

A arquitetura adota o padrão **Hexagonal (Ports & Adapters)**, expresso idiomaticamente via interfaces Go. A regra fundamental é: **a direção das dependências aponta sempre para o centro** (domínio/app).

```
┌─────────────────────────────────────────────────────────────────┐
│                         DELIVERY LAYER                          │
│              TUI (Bubble Tea)     │     CLI direto               │
└───────────────────────┬─────────────────────┘
                        │ chama
┌───────────────────────▼─────────────────────────────────────────┐
│                      APPLICATION LAYER                          │
│          Use Cases / Orchestration  (internal/app)              │
│   DaemonService │ EndpointService │ AnalysisService             │
│         define as PORTS (interfaces)                            │
└──────┬──────────────────┬──────────────────────┬────────────────┘
       │ implementado por  │ implementado por      │ implementado por
┌──────▼──────┐  ┌─────────▼────────┐  ┌──────────▼──────────────┐
│  COLLECTOR  │  │   AI ADAPTERS    │  │  STORAGE / REPORT       │
│  (pprof     │  │  ollama/         │  │  sqlite/ pdf/ filesystem│
│   HTTP)     │  │  openai/ (futuro)│  │                         │
└─────────────┘  └──────────────────┘  └─────────────────────────┘
                        │ todos dependem de
┌───────────────────────▼─────────────────────────────────────────┐
│                       DOMAIN LAYER                              │
│        Entities, Value Objects, Domain Errors                   │
│        (internal/domain) — zero dependências externas           │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Fluxo Principal de Dados

```
[Daemon Ticker]
      │
      ▼
[Collector] ── HTTP GET /debug/pprof/heap|allocs|goroutine|cpu|... ──► [Aplicação Go]
      │
      ▼ []ProfileResult (pprof binário parseado → representação texto)
[AIProvider.AnalyzeProfiles]
      │
      ├── prompt individual por perfil
      └── prompt consolidado (cross-profile)
      │
      ▼ AnalysisResult
[ReportWriter.Write] ── gera PDF ──► disco
      │
      ▼
[MetadataStore.SaveRun] ── persiste estado ──► SQLite
```

---

## 2. Estrutura do Projeto

```
pprof-analyzer/
├── cmd/
│   └── pprof-analyzer/
│       └── main.go              # Wire manual de todas as dependências
│
├── internal/
│   ├── domain/                  # Entidades e tipos centrais
│   │   ├── endpoint.go          # Endpoint, Credentials, Environment
│   │   ├── profile.go           # ProfileType, ProfileResult, CollectionRun
│   │   ├── analysis.go          # AnalysisResult, ProfileFinding, Recommendation, Severity
│   │   └── errors.go            # Erros de domínio (ErrEndpointUnreachable, etc.)
│   │
│   ├── app/                     # Use cases e definição dos Ports
│   │   ├── ports.go             # Interfaces: AIProvider, ProfileCollector, MetadataStore, ReportWriter, EndpointRepository
│   │   ├── daemon.go            # DaemonService: orquestra ciclos de coleta
│   │   ├── endpoint_service.go  # EndpointService: CRUD de endpoints
│   │   └── analysis_service.go  # AnalysisService: dispara análise e relatório
│   │
│   ├── collector/               # Adapter: coleta pprof via HTTP
│   │   ├── http_collector.go    # Implementa ProfileCollector
│   │   └── pprof_parser.go      # pprof binário → ProfileData texto
│   │
│   ├── ai/                      # Adapter: provedores de IA
│   │   ├── provider.go          # Interface interna + tipos de request/response
│   │   ├── ollama/
│   │   │   └── client.go        # Adapter Ollama (github.com/ollama/ollama/api)
│   │   └── langchain/           # Adapter multi-provider (fase 5)
│   │       └── client.go
│   │
│   ├── report/                  # Adapter: geração de PDF
│   │   ├── pdf_writer.go        # Implementa ReportWriter (maroto v2)
│   │   └── template.go          # Definição do template fixo do PDF
│   │
│   ├── storage/                 # Adapters: persistência em filesystem
│   │   ├── endpoint_repo.go     # Implementa EndpointRepository (endpoints.json)
│   │   ├── metadata_store.go    # Implementa MetadataStore (runs/{run-id}.json)
│   │   └── pprof_store.go       # Salva/deleta arquivos pprof com política de retenção
│   │
│   ├── config/                  # Carregamento e validação de configuração
│   │   ├── config.go            # Struct Config + Load()
│   │   └── defaults.go          # Valores padrão (modelo Ollama, intervalos, etc.)
│   │
│   └── tui/                     # Delivery: interface Bubble Tea
│       ├── app.go               # Model raiz, inicializa sub-models
│       ├── menu/
│       │   └── model.go         # Menu principal
│       ├── endpoints/
│       │   ├── list.go          # Listagem de endpoints
│       │   ├── form.go          # Formulário add/edit (charmbracelet/huh)
│       │   └── confirm.go       # Confirmação de remoção
│       ├── daemon/
│       │   └── model.go         # Start/Stop daemon
│       ├── dashboard/
│       │   └── model.go         # Estado atual em tempo real
│       ├── settings/
│       │   └── model.go         # Config Ollama + diretório de saída
│       └── styles/
│           └── styles.go        # Lip Gloss: cores, bordas, layout
│
├── Dockerfile
├── .goreleaser.yaml
├── Makefile
├── go.mod
└── go.sum
```

---

## 3. Camada de Domínio (`internal/domain`)

### 3.1 Entidades e Value Objects

```go
// endpoint.go
type Environment string
const (
    EnvProduction  Environment = "production"
    EnvStaging     Environment = "staging"
    EnvDevelopment Environment = "development"
)

type AuthType string
const (
    AuthNone        AuthType = "none"
    AuthBasic       AuthType = "basic"
    AuthBearerToken AuthType = "bearer"
)

type Credentials struct {
    AuthType AuthType
    Username string // basic auth
    Password string // basic auth
    Token    string // bearer token
}

type Endpoint struct {
    ID              string
    Name            string
    BaseURL         string
    Environment     Environment
    CollectInterval time.Duration
    Credentials     Credentials
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// profile.go
type ProfileType string
const (
    ProfileHeap         ProfileType = "heap"
    ProfileAllocs       ProfileType = "allocs"
    ProfileGoroutine    ProfileType = "goroutine"
    ProfileCPU          ProfileType = "profile"
    ProfileBlock        ProfileType = "block"
    ProfileMutex        ProfileType = "mutex"
    ProfileThreadCreate ProfileType = "threadcreate"
)

var AllProfileTypes = []ProfileType{
    ProfileHeap, ProfileAllocs, ProfileGoroutine,
    ProfileCPU, ProfileBlock, ProfileMutex, ProfileThreadCreate,
}

type ProfileData struct {
    Type      ProfileType
    RawPath   string        // caminho do arquivo pprof no disco
    TextSummary string      // top-N representação texto para o LLM
    CollectedAt time.Time
    SizeBytes int64
}

type CollectionRun struct {
    ID          string
    EndpointID  string
    StartedAt   time.Time
    CompletedAt time.Time
    Profiles    []ProfileData
    Status      RunStatus     // success, partial, failed
    FailureMsg  string
    ReportPath  string        // caminho do PDF gerado
}

// analysis.go
type Severity string
const (
    SeverityCritical Severity = "critical"
    SeverityWarning  Severity = "warning"
    SeverityNormal   Severity = "normal"
)

type ProfileFinding struct {
    ProfileType ProfileType
    Severity    Severity
    Summary     string
    Details     string
}

type Recommendation struct {
    Priority    int
    Title       string
    Description string
    CodeSuggestion string // opcional; trecho de código ou padrão
}

type AnalysisResult struct {
    RunID              string
    EndpointName       string
    Environment        Environment
    CollectedAt        time.Time
    OverallSeverity    Severity
    ExecutiveSummary   string
    PerProfileFindings []ProfileFinding
    ConsolidatedAnalysis string
    Recommendations    []Recommendation
    ModelUsed          string
    ToolVersion        string
}
```

### 3.2 Erros de Domínio

```go
// errors.go
var (
    ErrEndpointNotFound      = errors.New("endpoint not found")
    ErrEndpointUnreachable   = errors.New("endpoint unreachable")
    ErrProfileNotAvailable   = errors.New("profile not available on endpoint")
    ErrDaemonAlreadyRunning  = errors.New("daemon already running")
    ErrDaemonNotRunning      = errors.New("daemon not running")
    ErrAIProviderUnavailable = errors.New("AI provider unavailable")
)

type RetryExhaustedError struct {
    Endpoint string
    Attempts int
    Last     error
}
func (e *RetryExhaustedError) Error() string {
    return fmt.Sprintf("endpoint %s: all %d attempts failed: %v", e.Endpoint, e.Attempts, e.Last)
}
func (e *RetryExhaustedError) Unwrap() error { return e.Last }
```

---

## 4. Camada de Aplicação — Ports (`internal/app/ports.go`)

```go
// ProfileCollector coleta perfis pprof de um endpoint.
type ProfileCollector interface {
    Collect(ctx context.Context, endpoint domain.Endpoint) ([]domain.ProfileData, error)
}

// AIProvider analisa perfis e retorna diagnóstico estruturado.
type AIProvider interface {
    AnalyzeProfiles(ctx context.Context, req AnalysisRequest) (*domain.AnalysisResult, error)
}

// ReportWriter gera um PDF a partir de um AnalysisResult.
type ReportWriter interface {
    Write(ctx context.Context, result *domain.AnalysisResult) (filePath string, err error)
}

// EndpointRepository persiste e recupera endpoints cadastrados.
type EndpointRepository interface {
    List() ([]domain.Endpoint, error)
    Get(id string) (*domain.Endpoint, error)
    Save(e domain.Endpoint) error
    Delete(id string) error
}

// MetadataStore persiste estado de execuções para o dashboard.
type MetadataStore interface {
    SaveRun(run domain.CollectionRun) error
    GetLastRun(endpointID string) (*domain.CollectionRun, error)
    ListRuns(endpointID string, limit int) ([]domain.CollectionRun, error)
}

// PprofFileStore gerencia armazenamento e retenção de arquivos pprof no disco.
type PprofFileStore interface {
    Save(endpointID string, profileType domain.ProfileType, data []byte) (path string, err error)
    ApplyRetentionPolicy(endpointID string, profileType domain.ProfileType) error // mantém apenas as 3 últimas
}

// AnalysisRequest agrupa o contexto de entrada para o AIProvider.
type AnalysisRequest struct {
    Endpoint    domain.Endpoint
    CollectedAt time.Time
    Profiles    []domain.ProfileData
    ToolVersion string
}
```

---

## 5. Stack Tecnológica

### 5.1 Dependências Principais

| Categoria | Biblioteca | Versão | Justificativa |
|-----------|-----------|--------|---------------|
| **TUI** | `github.com/charmbracelet/bubbletea` | v1.x | Arquitetura Elm (puro, testável), padrão de facto em Go TUIs 2024/2025 |
| **TUI Estilo** | `github.com/charmbracelet/lipgloss` | v1.x | Styling declarativo; integra com Bubble Tea |
| **TUI Componentes** | `github.com/charmbracelet/bubbles` | v0.x | Table, TextInput, Spinner, Viewport prontos |
| **TUI Formulários** | `github.com/charmbracelet/huh` | v0.x | Forms idiomáticos dentro do modelo Bubble Tea; add/edit endpoints |
| **PDF** | `github.com/johnfercher/maroto/v2` | v2.x | API de alto nível; ativo; melhor ergonomia em Go |
| **JSON persistence** | `encoding/json` (stdlib) | — | Endpoints e metadados como arquivos JSON em disco |
| **Ollama Client** | `github.com/ollama/ollama/api` | latest | Client oficial; suporte nativo a tool calling |
| **pprof Parser** | `github.com/google/pprof/profile` | latest | Parsing canônico de arquivos pprof binários |
| **Config (JSON)** | `encoding/json` (stdlib) | — | Zero dependência externa; estrutura legível |
| **Logging** | `log/slog` (stdlib) | Go 1.21+ | Logging estruturado sem dependência externa |
| **Goroutine mgmt** | `github.com/oklog/run` | v1.x | Coordena múltiplas goroutines no daemon; qualquer uma saindo encerra as demais |
| **UUID** | `github.com/google/uuid` | v1.x | IDs de entidades e runs |
| **Testes** | `github.com/stretchr/testify` | v1.x | `assert` + `require`; padrão Go community |
| **Release** | `goreleaser/goreleaser` | v2.x | Cross-compilation + artefatos + Homebrew tap |

### 5.2 Dependências Futuras (Fase 5)

| Categoria | Biblioteca | Finalidade |
|-----------|-----------|------------|
| Multi-provider IA | `github.com/tmc/langchaingo` | Abstração unificada: OpenAI, Anthropic, Gemini |

### 5.3 Go Version

```
// go.mod
go 1.26
```

---

## 6. Configuração (`internal/config`)

### 6.1 Arquivo de Configuração (`~/.config/pprof-analyzer/config.json`)

```json
{
  "ollama": {
    "api_url": "http://localhost:11434",
    "model": "llama3.3:70b",
    "timeout_s": 120
  },
  "storage": {
    "data_dir": "~/.local/share/pprof-analyzer",
    "reports_dir": "~/pprof-reports"
  },
  "daemon": {
    "pid_file": "~/.local/share/pprof-analyzer/daemon.pid",
    "log_file": "~/.local/share/pprof-analyzer/daemon.log"
  },
  "collection": {
    "default_interval_s": 300,
    "cpu_sample_duration_s": 30,
    "max_retries": 3,
    "retry_backoff_s": 5
  }
}
```

### 6.2 Arquivo de Endpoints (`~/.local/share/pprof-analyzer/endpoints.json`)

```json
[
  {
    "id": "a1b2c3",
    "name": "api-gateway",
    "base_url": "http://localhost:6060",
    "environment": "production",
    "collect_interval_s": 300,
    "auth_type": "none",
    "created_at": "2026-04-01T10:00:00Z",
    "updated_at": "2026-04-01T10:00:00Z"
  }
]
```

Leitura/escrita com lock via `sync.RWMutex` no `EndpointRepository` para evitar race conditions quando o daemon e a TUI acessam simultaneamente.

---

## 7. Módulo Coletor (`internal/collector`)

### 7.1 `HTTPCollector`

```go
type HTTPCollector struct {
    httpClient *http.Client
    maxRetries int
    retryDelay time.Duration
}

func (c *HTTPCollector) Collect(ctx context.Context, endpoint domain.Endpoint) ([]domain.ProfileData, error)
```

**Fluxo por endpoint:**
1. Para cada `ProfileType` em `AllProfileTypes`, dispara coleta concorrente com `errgroup` (max 3 goroutines paralelas para não sobrecarregar o endpoint)
2. CPU profile (`/debug/pprof/profile?seconds=N`) é feito **por último** e de forma separada (latência de N segundos)
3. Cada coleta respeita `context.WithTimeout(ctx, perProfileTimeout)`
4. Em falha, retry com backoff exponencial até `maxRetries`; `RetryExhaustedError` sinaliza falha total
5. Se o endpoint retornar 404 para um perfil específico, registra como `ErrProfileNotAvailable` e continua os demais (sem abortar o ciclo)

### 7.2 `PprofParser` — Binário → Texto para LLM

```go
func ParseToTextSummary(rawData []byte, profileType domain.ProfileType) (string, error)
```

**Pipeline de conversão:**
1. `profile.Parse(bytes.NewReader(rawData))` → `*profile.Profile`
2. Calcula top-30 funções por flat value (memória/CPU/counts)
3. Serializa como tabela texto: `flat | flat% | cum | cum% | function`
4. Para `goroutine`: conta goroutines por estado + top-10 stacks únicas
5. Trunca para **4.000 tokens** máximos por perfil para não explodir o context window

---

## 8. Módulo de IA (`internal/ai`)

### 8.1 Adapter Ollama (`internal/ai/ollama`)

```go
type OllamaClient struct {
    client *api.Client
    model  string
    timeout time.Duration
}

func (c *OllamaClient) AnalyzeProfiles(ctx context.Context, req app.AnalysisRequest) (*domain.AnalysisResult, error)
```

**Estratégia de análise (2 fases):**

**Fase 1 — Análise Individual:**
Para cada perfil coletado, envia um prompt com:
- System prompt: tipo de perfil, unidades, instrução de output JSON estruturado
- User prompt: top-30 tabela texto + call tree top-5 caminhos mais quentes
- Output esperado: `ProfileFinding` (severity + summary + details)

**Fase 2 — Análise Consolidada:**
Envia os `ProfileFinding` individuais juntos com um prompt de correlação:
- Identifica padrões cross-profile (ex: heap crescendo + goroutine leak juntos)
- Gera `ConsolidatedAnalysis`, `Recommendations`, `OverallSeverity`

**Tool Calling (quando o modelo suporta):**
```
Tool: get_top_functions(profile_id, n, sort_by)
Tool: get_goroutine_summary(profile_id)
Tool: get_call_path(function_name, depth)
```
Permite que o modelo "navegue" no perfil em vez de receber tudo de uma vez.

**Detecção de capacidade:**
```go
resp, _ := client.Show(ctx, &api.ShowRequest{Model: model})
supportsTools := slices.Contains(resp.Capabilities, "tools")
```

### 8.2 Prompts

**System prompt (heap):**
```
You are a Go performance engineer. You are analyzing a Go heap profile.
Units: flat = bytes currently allocated (retained); cum = bytes in call chain.
flat% = percentage of total heap; cum% = inclusive percentage.
Return ONLY valid JSON matching the schema provided. No prose outside JSON.
```

**Output schema solicitado ao modelo:**
```json
{
  "severity": "critical|warning|normal",
  "summary": "one sentence",
  "details": "2-4 sentences of technical analysis",
  "recommendations": [
    {
      "priority": 1,
      "title": "short title",
      "description": "actionable description",
      "code_suggestion": "optional Go code snippet"
    }
  ]
}
```

### 8.3 Modelo Padrão Recomendado

| Prioridade | Modelo | Motivo |
|-----------|--------|--------|
| 1º | `llama3.3:70b` | Melhor raciocínio open-source sobre código; tool calling nativo |
| 2º | `qwen2.5-coder:32b` | Forte em Go; tool calling confiável; menor que 70B |
| 3º | `llama3.2:3b` | Edge/baixo recurso; análise menos profunda |

---

## 9. Módulo de Relatório PDF (`internal/report`)

### 9.1 Biblioteca: `maroto/v2`

Usando o builder pattern do maroto v2:

```go
m := core.NewMaroto(props.Maroto{
    PageSize:    props.A4,
    Orientation: props.Portrait,
})
```

### 9.2 Estrutura do PDF (Template Fixo)

| Seção | Conteúdo |
|-------|----------|
| **Cabeçalho** | Logo (opcional) · Nome da aplicação · Ambiente · Data/hora · Intervalo |
| **Sumário Executivo** | Badge de severidade colorido · Parágrafo do `ExecutiveSummary` |
| **Análise por Perfil** | Uma sub-seção por `ProfileFinding`; tabela de stats resumidos |
| **Análise Consolidada** | Texto do `ConsolidatedAnalysis`; diagrama textual de correlações se disponível |
| **Recomendações** | Tabela numerada por prioridade; blocos de código quando `CodeSuggestion` não-vazio |
| **Rodapé** | Versão da ferramenta · Modelo de IA utilizado · Timestamp geração |

**Paleta de severidade:**
- Crítico: `#DC2626` (vermelho)
- Atenção: `#D97706` (âmbar)
- Normal: `#16A34A` (verde)

### 9.3 Nomenclatura dos Arquivos

```
{reports_dir}/{app-name}/{env}/{YYYY-MM-DD}/{app-name}_{env}_{YYYYMMDD_HHMMSS}.pdf
```

---

## 10. Daemon (`internal/app/daemon.go`)

### 10.1 Estratégia de Background Process

O daemon é iniciado como um processo separado via `exec.Command` com re-exec do próprio binário. `SysProcAttr` é definido em arquivos específicos por plataforma via build tags para isolar o código de plataforma.

```
pprof-analyzer daemon start
  └── verifica se PID file existe e processo está ativo (kill(pid, 0))
  └── se não: exec.Command(os.Args[0], "--daemon-internal")
        └── Linux/macOS: SysProcAttr{Setsid: true}  → nova session, sem SIGHUP
        └── Windows:     SysProcAttr{CreationFlags: DETACHED_PROCESS}
  └── cmd.Stdout/Stderr → log file; cmd.Stdin → nil
  └── cmd.Start() sem cmd.Wait() → retorna ao terminal

pprof-analyzer daemon stop
  └── lê PID file
  └── Linux/macOS: SIGTERM → aguarda 10s → SIGKILL se necessário
  └── Windows: os.Process.Kill()
  └── remove PID file
```

**Arquivos de build tag:**
```
internal/daemon/proc_unix.go   //go:build !windows
internal/daemon/proc_windows.go //go:build windows
```

### 10.2 Loop Principal

Usa `oklog/run` para coordenar goroutines: se qualquer uma sair (erro ou shutdown), todas as demais são interrompidas via interrupt:

```go
func (d *DaemonService) Run(ctx context.Context) error {
    var g run.Group

    // Uma goroutine por endpoint
    for _, ep := range endpoints {
        ep := ep // capture
        ctx, cancel := context.WithCancel(ctx)
        g.Add(
            func() error { return d.runEndpointLoop(ctx, ep) },
            func(err error) { cancel() },
        )
    }

    // Goroutine de signal/ctx cancellation
    g.Add(
        func() error { <-ctx.Done(); return ctx.Err() },
        func(err error) {},
    )

    return g.Run()
}

func (d *DaemonService) runEndpointLoop(ctx context.Context, ep domain.Endpoint) {
    ticker := time.NewTicker(ep.CollectInterval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            cycleCtx, cancel := context.WithTimeout(ctx, ep.CollectInterval/2)
            d.runCollectionCycle(cycleCtx, ep)
            cancel()
        }
    }
}
```

### 10.3 Política de Retenção

Executada pelo `PprofFileStore` após cada ciclo de coleta bem-sucedido:
1. Lista arquivos para `(endpointID, profileType)` ordenados por timestamp
2. Se `len(files) > 3`, deleta os mais antigos até manter exatamente 3
3. Operação é idempotente e não bloqueia o ciclo principal

---

## 11. Interface TUI (`internal/tui`)

### 11.1 Bubble Tea — Arquitetura Elm

Cada tela é um `tea.Model` com os três métodos: `Init()`, `Update(msg)`, `View()`.

A model raiz (`app.go`) gerencia qual tela está ativa e faz composição:

```go
type AppModel struct {
    currentScreen Screen
    menu          menu.Model
    endpointsList endpoints.ListModel
    endpointForm  endpoints.FormModel
    dashboard     dashboard.Model
    settings      settings.Model
    daemonCtrl    daemon.Model
    // serviços injetados
    endpointSvc *app.EndpointService
    daemonSvc   *app.DaemonService
}
```

### 11.2 Fluxo de Navegação

```
AppModel (root)
├── MenuScreen          → menu principal com seleção por setas
│   ├── → EndpointsListScreen
│   │     ├── → EndpointFormScreen (add)
│   │     ├── → EndpointFormScreen (edit)  
│   │     └── → EndpointConfirmDeleteScreen
│   ├── → DaemonScreen  (start/stop com status em tempo real)
│   ├── → DashboardScreen (auto-refresh a cada 5s via tea.Tick)
│   └── → SettingsScreen
└── ESC / q = volta para o menu pai
```

### 11.3 Dashboard — Refresh em Tempo Real

O dashboard usa `tea.Tick` para se auto-atualizar a cada 5 segundos. Busca dados do `MetadataStore` (SQLite) para evitar leitura de arquivo ao vivo:

```
┌─ pprof-analyzer dashboard ──────────────────────────────────────┐
│  Daemon: ● RUNNING                               [atualizado: 14:23:05] │
├──────────────┬─────────────┬───────────────┬────────────────────┤
│  Aplicação   │  Ambiente   │  Última Coleta│  Alertas           │
├──────────────┼─────────────┼───────────────┼────────────────────┤
│  api-gateway │  production │  14:20:01     │  ⚠ 2 (atenção)    │
│  worker-svc  │  staging    │  14:18:45     │  ✓ 0              │
│  auth-api    │  production │  ERRO         │  — sem dados —     │
└──────────────┴─────────────┴───────────────┴────────────────────┘
│  [q] sair  [r] atualizar  [↑↓] navegar                         │
└─────────────────────────────────────────────────────────────────┘
```

---

## 12. Armazenamento — Filesystem

### 12.1 Estrutura de Diretórios

```
{data_dir}/
├── endpoints.json                          # lista de endpoints cadastrados
├── daemon.pid                              # PID do processo daemon
├── daemon.log                              # log do daemon
│
├── pprof/                                  # arquivos pprof coletados
│   └── {endpoint-id}/
│       └── {profile-type}/
│           ├── 20260401_142000.pb.gz       # timestamp como nome
│           ├── 20260401_142500.pb.gz
│           └── 20260401_143000.pb.gz       # máximo 3 arquivos (retenção)
│
└── runs/                                   # metadados de cada ciclo de coleta
    └── {endpoint-id}/
        ├── 20260401_142000.json
        └── 20260401_142500.json

{reports_dir}/
└── {app-name}/
    └── {environment}/
        └── {YYYY-MM-DD}/
            └── {app-name}_{env}_{YYYYMMDD_HHMMSS}.pdf
```

### 12.2 Formato de Run Metadata (`runs/{endpoint-id}/{timestamp}.json`)

```json
{
  "id": "run-uuid",
  "endpoint_id": "a1b2c3",
  "started_at": "2026-04-01T14:20:00Z",
  "completed_at": "2026-04-01T14:20:35Z",
  "status": "success",
  "failure_msg": "",
  "report_path": "~/pprof-reports/api-gateway/production/2026-04-01/api-gateway_production_20260401_142000.pdf",
  "profiles": [
    { "type": "heap", "raw_path": "...", "size_bytes": 45312, "collected_at": "..." },
    { "type": "goroutine", "raw_path": "...", "size_bytes": 8192, "collected_at": "..." }
  ]
}
```

### 12.3 Escrita Atômica de Arquivos

Todos os writes em arquivos JSON críticos (`endpoints.json`, run metadata) usam o padrão write-to-temp + rename para evitar corrupção em caso de crash:

```go
func atomicWriteJSON(path string, v any) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil { return err }
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0600); err != nil { return err }
    return os.Rename(tmp, path)  // atômico no mesmo filesystem
}
```

### 12.4 Leitura do Dashboard

O `MetadataStore` carrega o run mais recente de cada endpoint lendo apenas o arquivo JSON com maior timestamp no diretório `runs/{endpoint-id}/` — sem varredura completa. O `EndpointRepository` usa `sync.RWMutex` para acesso concorrente seguro ao `endpoints.json`.

---

## 13. Injeção de Dependências (`cmd/pprof-analyzer/main.go`)

Manual, explícita e legível — sem framework:

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    cfg, err := config.Load()
    if err != nil { slog.Error("config load failed", "err", err); os.Exit(1) }

    endpointRepo := storage.NewEndpointRepository(cfg.Storage.DataDir)
    metaStore    := storage.NewMetadataStore(cfg.Storage.DataDir)
    pprofStore   := storage.NewPprofStore(cfg.Storage.DataDir)
    collector    := collector.NewHTTPCollector(cfg.Collection.MaxRetries, cfg.Collection.RetryBackoff)
    aiProvider   := ollama.NewClient(cfg.Ollama.APIURL, cfg.Ollama.Model, cfg.Ollama.Timeout)
    reportWriter := report.NewPDFWriter(cfg.Storage.ReportsDir)

    endpointSvc := app.NewEndpointService(endpointRepo)
    analysisSvc := app.NewAnalysisService(collector, aiProvider, reportWriter, pprofStore, metaStore)
    daemonSvc   := app.NewDaemonService(endpointSvc, analysisSvc, cfg.Daemon.PIDFile, cfg.Daemon.LogFile)

    // Verifica se está rodando em modo daemon (flag interna)
    if len(os.Args) > 1 && os.Args[1] == "--daemon-internal" {
        if err := daemonSvc.RunInProcess(ctx); err != nil && !errors.Is(err, context.Canceled) {
            slog.Error("daemon error", "err", err)
            os.Exit(1)
        }
        return
    }

    // Modo normal: TUI
    m := tui.New(endpointSvc, daemonSvc, metaStore, cfg)
    if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
        slog.Error("tui error", "err", err)
        os.Exit(1)
    }
}
```

---

## 14. Testes

### 14.1 Estratégia

| Módulo | Tipo de Teste | Ferramentas |
|--------|--------------|-------------|
| `internal/domain` | Unitário puro | stdlib `testing` |
| `internal/app` | Unitário com mocks manuais das ports | `testify` + mocks hand-written |
| `internal/collector` | Integração com `httptest.NewServer` | `testify` + `httptest` |
| `internal/storage` | Integração com filesystem temp | `testify` + `t.TempDir()` |
| `internal/report` | Golden file (compara PDF gerado com fixture) | `testify` |
| `internal/ai/ollama` | Integração com Ollama mock HTTP | `testify` + `httptest` |
| `internal/tui` | Unitário do modelo Elm (`Update()` puro) | `testify` + `bubbletea/testprogram` |
| Retenção de arquivos | Integração com filesystem temp | `testify` + `t.TempDir()` |

### 14.2 Convenção de Mocks

Mocks hand-written para as interfaces de ports (pequenas, máx. 5 métodos cada):

```go
// internal/app/testhelpers_test.go
type mockAIProvider struct {
    analyzeFunc func(ctx context.Context, req AnalysisRequest) (*domain.AnalysisResult, error)
}
func (m *mockAIProvider) AnalyzeProfiles(ctx context.Context, req AnalysisRequest) (*domain.AnalysisResult, error) {
    return m.analyzeFunc(ctx, req)
}
```

---

## 15. Distribuição

### 15.1 GoReleaser v2 (`.goreleaser.yaml`)

```yaml
version: 2
builds:
  - id: pprof-analyzer
    main: ./cmd/pprof-analyzer
    binary: pprof-analyzer
    env:
      - CGO_ENABLED=0   # zero CGO: todas as dependências são pure Go
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}
      - -X main.buildDate={{.Date}}

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

dockers:
  - image_templates:
      - "ghcr.io/{{.Env.GITHUB_REPOSITORY_OWNER}}/pprof-analyzer:{{.Tag}}"
      - "ghcr.io/{{.Env.GITHUB_REPOSITORY_OWNER}}/pprof-analyzer:latest"
    dockerfile: Dockerfile
    build_flag_templates:
      - "--platform=linux/amd64"
```

### 15.2 Dockerfile

```dockerfile
FROM gcr.io/distroless/static-debian12:nonroot
COPY pprof-analyzer /usr/local/bin/pprof-analyzer
ENTRYPOINT ["pprof-analyzer"]
```

Build multi-stage: `golang:1.24-alpine` para compilar, `distroless/static` para imagem final (~5MB).

### 15.3 Canais de Distribuição

| Canal | Método |
|-------|--------|
| GitHub Releases | GoReleaser (automático no push de tag `v*`) |
| `go install` | `go install github.com/{owner}/pprof-analyzer/cmd/pprof-analyzer@latest` |
| Docker / GHCR | GoReleaser docker config |
| Homebrew tap | `brew tap {owner}/tap && brew install pprof-analyzer` (fase 2) |

---

## 16. Segurança e Considerações

| Tema | Decisão |
|------|---------|
| Credenciais em disco | Armazenadas em plaintext no `endpoints.json`. README deve documentar claramente. Criptografia de credenciais é escopo de fase futura. |
| Permissão do arquivo de config | `chmod 600` aplicado automaticamente no primeiro save |
| Sem servidor HTTP | A ferramenta não expõe nenhuma porta de rede; elimina superfície de ataque |
| Ollama local | Sem tráfego de dados de perfil para nuvem no MVP; privacidade por design |
| Path traversal | `filepath.Clean` + validação de que o path final está dentro de `DataDir` antes de qualquer write/read |
| Acesso concorrente a JSON | `sync.RWMutex` no `EndpointRepository` e `MetadataStore` para garantir integridade dos arquivos |

---

## 17. Roadmap Técnico por Fase

| Fase | Spec | Mudanças de Arquitetura |
|------|------|------------------------|
| **MVP** | Coletor, Ollama, PDF, TUI, Daemon, SQLite | — baseline deste documento |
| **Fase 2** (Análise Temporal) | Comparação entre runs | Novo use case `CompareRunsService`; queries temporais no SQLite |
| **Fase 3** (Notificações) | Email, Slack, Webhook | Nova port `NotificationSender`; adapters por canal |
| **Fase 4** (Web UI) | Dashboard web, viewer de PDF | Nova delivery layer HTTP; `internal/api` com handler REST |
| **Fase 5** (Multi-provider) | OpenAI, Anthropic, Gemini | Adicionar adapters em `internal/ai/`; migrar para `langchaingo` ou adapters diretos; zero mudança em `internal/app` |

---

## 18. Checklist de Início

- [ ] Criar repositório Git com estrutura de pastas definida na seção 2
- [ ] Inicializar `go.mod` com Go 1.26
- [ ] Adicionar dependências com `go get`
- [ ] Implementar domain layer (zero dependências)
- [ ] Implementar ports (interfaces apenas)
- [ ] Implementar `HTTPCollector` + `PprofParser` com testes via `httptest`
- [ ] Implementar `OllamaClient` com prompt engineering definido
- [ ] Implementar `PDFWriter` com template fixo
- [ ] Implementar TUI com Bubble Tea (começar pelo menu e endpoint form)
- [ ] Implementar lógica de daemon (PID file + fork)
- [ ] Integrar tudo no `main.go` com DI manual
- [ ] Configurar GoReleaser + Dockerfile
- [ ] Escrever README com requisitos (Ollama instalado, modelo compatível)
