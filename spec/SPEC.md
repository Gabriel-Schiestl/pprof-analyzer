# pprof-analyzer — Especificação do Projeto

> Versão 1.0 — Abril 2026

---

## 1. Visão Geral

**pprof-analyzer** é uma ferramenta de monitoramento contínuo e análise inteligente de perfis pprof de aplicações Go. Ela coleta automaticamente os endpoints pprof de aplicações cadastradas em intervalos configuráveis, armazena os arquivos gerados e aciona um agente de IA local (via Ollama) para analisar cada perfil individualmente e em conjunto, produzindo relatórios em PDF com diagnósticos e sugestões de correção.

### Motivação

A dor original surgiu de um incidente de OOM (Out of Memory) em produção, onde a análise manual de arquivos pprof se mostrou lenta, repetitiva e propensa a erros humanos. A ferramenta automatiza esse ciclo de coleta → análise → relatório, permitindo identificar memory leaks, goroutine leaks e outros problemas de performance antes que virem incidentes.

### Público-alvo

- Desenvolvedores Go que monitoram suas próprias aplicações
- Times de SRE/DevOps que precisam de visibilidade contínua sobre saúde de aplicações
- A ferramenta será open source e distribuída para a comunidade

---

## 2. Casos de Uso Principais

| # | Caso de Uso | Descrição |
|---|-------------|-----------|
| UC01 | Cadastrar endpoint | Usuário registra uma aplicação com URL base, nome, ambiente e credenciais opcionais |
| UC02 | Iniciar monitoramento | Usuário inicia o daemon; a ferramenta começa a coletar pprof nos intervalos configurados |
| UC03 | Parar monitoramento | Usuário para o daemon via comando explícito |
| UC04 | Visualizar estado atual | Usuário consulta um dashboard TUI mostrando aplicações, última coleta e alertas |
| UC05 | Consultar relatórios | Usuário acessa os PDFs gerados para cada coleta |
| UC06 | Gerenciar endpoints | Usuário lista, edita ou remove endpoints cadastrados |

---

## 3. Funcionalidades

### 3.1 Gerenciamento de Endpoints

Cada endpoint cadastrado contém:

- **Nome** — identificador legível da aplicação (ex: `api-gateway`)
- **URL base** — endereço raiz da aplicação (ex: `http://localhost:6060`)
- **Ambiente** — ex: `production`, `staging`, `development`
- **Intervalo de coleta** — tempo em segundos entre cada ciclo de coleta (mesmo intervalo para todos os perfis)
- **Credenciais** (opcional) — para endpoints protegidos (ex: Basic Auth, token de header); endpoints públicos não exigem

Os endpoints são persistidos em um arquivo JSON local.

### 3.2 Perfis pprof Coletados

A cada ciclo, a ferramenta coleta **todos** os perfis pprof disponíveis no endpoint:

- `heap` — alocações de memória em uso
- `allocs` — histórico total de alocações
- `goroutine` — goroutines ativas e suas stacks
- `cpu` (profile) — consumo de CPU durante um intervalo de amostragem
- `block` — operações bloqueadas em primitivas de sincronização
- `mutex` — contenção de mutexes
- `threadcreate` — threads do sistema operacional criadas

### 3.3 Coleta e Armazenamento

- O daemon roda em background após ser iniciado via comando
- A coleta é automática e silenciosa — sem interação do usuário
- Arquivos pprof são armazenados localmente em disco, organizados por aplicação, perfil e timestamp
- **Política de retenção:** a cada 3 análises concluídas para um mesmo endpoint + perfil, o arquivo mais antigo é excluído
- **Tratamento de falhas:** se um endpoint estiver inacessível, a ferramenta tenta 3 vezes antes de desistir; se todas as tentativas falharem, a coleta é ignorada e nenhum relatório é gerado para aquele ciclo

### 3.4 Análise por Agente de IA

Após cada coleta bem-sucedida, um agente de IA é acionado para analisar os arquivos. O fluxo é:

1. **Relatório individual** — o agente analisa cada perfil separadamente e gera uma seção do relatório
2. **Relatório consolidado** — o agente cruza os perfis coletados no mesmo ciclo e identifica correlações, padrões anômalos, possíveis leaks e gargalos

O agente também **sugere correções** — trecho de código, padrão de design, ou mudança de configuração — quando identifica um problema concreto.

A análise é **silenciosa e automática** — sem chat ou interação do usuário nesta versão.

#### Integração com IA

- **Provedor inicial:** Ollama (instância configurada pelo próprio usuário; a ferramenta consome a API)
- **Modelo padrão:** configurável, mas com um modelo padrão pré-definido que suporte uso de *tools/function calling*
- **Extensibilidade:** a arquitetura deve permitir integração futura com outros provedores (OpenAI, Anthropic, Gemini, etc.) sem reescrita do core

### 3.5 Geração de Relatório PDF

Cada ciclo de coleta bem-sucedido resulta em um PDF com template fixo, contendo:

1. **Cabeçalho** — nome da aplicação, ambiente, data/hora da coleta, intervalo configurado
2. **Sumário Executivo** — visão geral dos achados com severidade (crítico / atenção / normal)
3. **Análise por Perfil** — uma seção por perfil coletado com os achados individuais
4. **Análise Consolidada** — correlações entre perfis, tendências observadas, diagnóstico geral
5. **Recomendações** — lista priorizada de ações sugeridas, incluindo sugestões de correção de código quando aplicável
6. **Rodapé** — versão da ferramenta, modelo de IA utilizado

Os PDFs são armazenados localmente e consultados diretamente pelo usuário fora da ferramenta (não há viewer embutido no MVP).

---

## 4. Interface — CLI / TUI

A interface inicial é uma **TUI interativa** (menu de seleção no terminal). O usuário navega por menus e não precisa memorizar subcomandos.

### Fluxo de navegação (MVP)

```
pprof-analyzer
└── Menu Principal
    ├── Endpoints
    │   ├── Listar endpoints
    │   ├── Adicionar endpoint
    │   ├── Editar endpoint
    │   └── Remover endpoint
    ├── Daemon
    │   ├── Iniciar monitoramento
    │   └── Parar monitoramento
    ├── Dashboard
    │   └── Estado atual (aplicações, última coleta, alertas)
    └── Configurações
        ├── Configurar Ollama (URL da API, modelo)
        └── Diretório de saída dos relatórios
```

### Comportamento do Daemon

- Iniciado via menu ou comando direto
- Roda em background — o terminal fica livre após o start
- Persiste até ser parado explicitamente via comando/menu ou até o sistema ser desligado
- Não requer que o terminal esteja aberto para continuar rodando

---

## 5. Dashboard (TUI)

O dashboard exibe, em tempo real ou sob demanda:

- Lista de endpoints cadastrados
- Status de cada endpoint: `ativo`, `erro`, `sem coleta`
- Timestamp da última coleta bem-sucedida
- Número de alertas no último relatório (se houver)
- Status do daemon: `rodando` / `parado`

---

## 6. Distribuição e Implantação

A ferramenta deve funcionar em dois modos de uso:

| Modo | Descrição |
|------|-----------|
| **Local** | Usuário baixa o binário e roda no próprio terminal |
| **Container** | Imagem Docker disponível para rodar em servidor |

A configuração (endpoints, credenciais, diretório de saída, Ollama) é feita via arquivo local (JSON ou similar) e/ou pela própria TUI.

---

## 7. Roadmap — Fases

### Fase 1 — MVP
- Cadastro de endpoints via TUI
- Coleta automática de todos os perfis pprof
- Armazenamento de arquivos pprof com política de retenção (3 coletas)
- Análise por agente Ollama (análise individual + consolidada)
- Geração de relatório PDF com template fixo
- Daemon em background (start/stop via TUI)
- Dashboard TUI básico
- Suporte a Ollama com modelo configurável

### Fase 2 — Análise Temporal
- Análise de tendências entre coletas (crescimento de heap, goroutine drift, etc.)
- Relatório comparativo entre snapshots

### Fase 3 — Notificações
- Notificação de novos relatórios via email, Slack ou webhook configurável
- Alertas de anomalia (ex: heap cresceu X% entre coletas)

### Fase 4 — Interface Web
- Dashboard web com histórico visual
- Visualização de relatórios no browser
- Gerenciamento de endpoints via UI web

### Fase 5 — Multi-provider IA
- Suporte a OpenAI, Anthropic (Claude), Google Gemini
- Seleção de provedor e modelo via configuração

---

## 8. Restrições e Premissas

- A ferramenta **não gerencia** a instância Ollama — assume que o usuário tem uma rodando e acessível
- Endpoints pprof devem seguir o padrão do pacote `net/http/pprof` do Go
- No MVP, **não há consulta de histórico de relatórios** dentro da ferramenta — os PDFs são consultados diretamente no sistema de arquivos
- Credenciais são armazenadas localmente; segurança do arquivo de configuração é responsabilidade do usuário
- A análise de CPU profile pode ter latência maior que outros perfis (requer período de amostragem)

---

## 9. Glossário

| Termo | Definição |
|-------|-----------|
| pprof | Formato de perfil de performance do Go, gerado pelo pacote `net/http/pprof` |
| Endpoint | URL base de uma aplicação Go com o servidor pprof exposto |
| Ciclo de coleta | Uma rodada de coleta de todos os perfis de um endpoint |
| Daemon | Processo em background responsável por disparar as coletas nos intervalos configurados |
| Relatório | Documento PDF gerado pelo agente de IA após cada ciclo de coleta |
| Agente de IA | Instância de modelo LLM (via Ollama) que analisa os perfis e gera os relatórios |
