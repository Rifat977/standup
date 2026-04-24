# standup — Daily Standup CLI
### Project plan & implementation guide for Claude Code
> **Stack:** Go · Bubbletea TUI · GitHub API · OpenAI GPT-4o

---

## 1. Project Overview

`standup` is a terminal-based CLI tool that automatically collects your last 12 hours of git commits and GitHub pull requests, lets you add your plans for the day and any blockers, then uses OpenAI GPT-4o to generate a clean, human-sounding standup summary — all from your terminal in under 30 seconds.

**Core value proposition:** no more staring at a blank standup form. The tool does the heavy lifting — you just review and share.

### Key features

- Pulls real git commits from local repos (last 12h, configurable)
- Fetches GitHub PRs — title, review status, CI state, age
- Editable "Today" and "Blockers" sections in a clean TUI
- One keypress to generate AI summary via OpenAI API (streamed live)
- Copy to clipboard or post directly to Slack
- Local history of past standups (last 30 days)
- In-TUI config editor — set tokens, repos, and preferences without leaving the terminal
- `--plain` flag for piping output to other tools or CI scripts

---

## 2. Architecture & Data Flow

```
run standup → read config → git log (12h) → GitHub API (PRs) → TUI opens → press S → OpenAI → AI summary → copy / Slack
```

### Layer breakdown

| Layer      | Package                | Responsibility |
|------------|------------------------|----------------|
| Git        | `internal/git`         | Run `git log --since=12h`, parse commits per repo, group by author |
| GitHub     | `internal/github`      | Fetch open/merged PRs via REST API, filter by last 12h, include CI status |
| AI         | `internal/ai`          | Build prompt from commits + PRs + notes, call OpenAI chat completions, stream response token by token |
| UI         | `internal/ui`          | Bubbletea root model, tab routing, keyboard shortcuts, all 4 views |
| Formatter  | `internal/formatter`   | Render standup as Markdown, plain text, or Slack mrkdwn |
| Share      | `internal/share`       | Copy to system clipboard (pbcopy/xclip/wl-copy), POST to Slack webhook |
| History    | `internal/history`     | Save/load `~/.standup/history.json`, keep last 30 entries |
| Config     | `internal/config`      | Load `~/.standup/config.yaml`, apply env overrides, expose defaults |

---

## 3. TUI Screens

The TUI has **4 tabs**, switchable with keys `1`–`4` at any time. The app opens on tab 1.

---

### [1] Main view — standup

The primary screen. Shown on launch. Contains three sections:

- **Commits & PRs (last 12h)** — auto-populated, read-only, grouped by repo
- **Today** — editable text field, press `e` to enter edit mode
- **Blockers** — editable text field, press `Tab` to switch between fields

Press `s` to send everything to OpenAI and switch to the Summary tab.

```
┌─ standup  [1]main [2]summary [3]history [4]config ─── Apr 25 ─┐
│                                                                 │
│  Commits & PRs  (last 12h)                                     │
│  ── api-service ──────────────────────────────────────────     │
│    a3f9c2b  feat: add JWT refresh token rotation               │
│    91ec44d  fix: null pointer in middleware handler            │
│  Pull Requests                                                  │
│    #421  feat: oauth2 refresh   APPROVED   CI pass   2h ago   │
│    #418  fix: parser panic      IN REVIEW  CI pass   4h ago   │
│                                                                 │
│  Today  (press e to edit)                                      │
│    > Continue oauth2 work, review #418                        │
│  Blockers                                                       │
│    > none                                                       │
├─────────────────────────────────────────────────────────────────┤
│  e:edit  s:AI summary  c:copy  r:refresh  1-4:tabs  q:quit    │
└─────────────────────────────────────────────────────────────────┘
```

---

### [2] Summary view — AI output

Shows the OpenAI-generated standup note, streamed live as tokens arrive. User can edit the output after generation, copy it, or post to Slack.

- Press `r` to regenerate with a fresh API call
- Press `e` to edit the generated text before sharing
- Token count is shown bottom-right for cost awareness

```
┌─ standup  [1]main [2]summary [3]history [4]config ─────────────┐
│                                                                 │
│  AI Standup Summary                     model: gpt-4o          │
│  ─────────────────────────────────────────────────────────     │
│  Yesterday                                                      │
│  Worked on the API service — implemented JWT refresh token     │
│  rotation and fixed a null pointer bug in the middleware       │
│  handler. Also bumped Go to 1.22 in the infra repo.           │
│                                                                 │
│  Today                                                          │
│  Continuing oauth2 work and planning to review PR #418.       │
│                                                                 │
│  Blockers                                                       │
│  None at the moment.                                           │
│                                                                 │
│  ● generating...                        tokens used: 312       │
├─────────────────────────────────────────────────────────────────┤
│  e:edit  c:copy  s:post to Slack  r:regenerate  q:quit        │
└─────────────────────────────────────────────────────────────────┘
```

---

### [3] History view

Two-pane layout: left side shows a list of past dates, right side shows the full standup for the selected date.

- Navigate with `↑↓` or `j/k`
- Press `c` to copy selected entry to clipboard
- History stored at `~/.standup/history.json` (last 30 entries)

---

### [4] Config view

Inline editor for all settings — no need to manually edit YAML. Shows current values and lets you update them without leaving the TUI.

- GitHub token and target repos
- OpenAI API key and model selection (`gpt-4o` / `gpt-4o-mini`)
- Scan directories for git repos
- Slack webhook URL and channel
- Time window (default `12h`)

---

## 4. File Structure

> All internal packages are independent with no circular dependencies. The `ui` package imports everything else; nothing imports `ui`.

```
standup/
├── cmd/standup/
│   └── main.go                  ← cobra CLI entry: run, init, show
│
├── internal/
│   ├── config/
│   │   └── config.go            ← load/save ~/.standup/config.yaml
│   │
│   ├── git/
│   │   └── scanner.go           ← git log --since=Xh, parse commits per repo
│   │
│   ├── github/
│   │   └── client.go            ← fetch PRs via REST API (last 12h window)
│   │
│   ├── ai/
│   │   └── summarize.go         ← build prompt + call OpenAI + stream response
│   │
│   ├── ui/
│   │   ├── model.go             ← root bubbletea model, tab routing
│   │   ├── styles.go            ← lipgloss color/layout tokens
│   │   ├── main_view.go         ← [1] commits + PRs + editable today/blockers
│   │   ├── summary_view.go      ← [2] streamed AI summary with edit mode
│   │   ├── history_view.go      ← [3] two-pane history browser
│   │   └── config_view.go       ← [4] inline config editor
│   │
│   ├── formatter/
│   │   └── format.go            ← markdown, plain text, slack mrkdwn output
│   │
│   ├── history/
│   │   └── history.go           ← save/load ~/.standup/history.json
│   │
│   └── share/
│       └── share.go             ← clipboard (pbcopy/xclip/wl-copy) + Slack POST
│
├── config.example.yaml          ← template config with comments
├── go.mod
├── go.sum
└── README.md
```

---

## 5. Config File

Stored at `~/.standup/config.yaml`. Created automatically on first run via `standup init`. All fields can also be overridden with environment variables (e.g. `OPENAI_API_KEY`).

```yaml
# ~/.standup/config.yaml

author: "your-git-name"         # filters git commits by this author
since: 12h                      # time window: 12h, 24h, 48h

scan_dirs:                      # directories to scan for git repos
  - ~/projects
  - ~/work

github:
  token: "ghp_xxxxxxxxxxxx"     # Personal Access Token (repo scope)
  repos:                        # repos to fetch PRs from
    - owner/repo-one
    - owner/repo-two

openai:
  api_key: "sk-xxxxxxxxxxxx"
  model: gpt-4o                 # or gpt-4o-mini for faster/cheaper
  max_tokens: 500

slack:
  webhook_url: ""               # https://hooks.slack.com/services/...
  channel: "#standup"

format: markdown                # markdown | plain | slack
```

---

## 6. Tech Stack

| Package       | Import path                      | Purpose |
|---------------|----------------------------------|---------|
| Bubbletea     | `charmbracelet/bubbletea`        | TUI framework (Elm architecture for terminal) |
| Lipgloss      | `charmbracelet/lipgloss`         | Terminal colors, borders, layout |
| Bubbles       | `charmbracelet/bubbles`          | Text inputs, spinner, viewport component |
| go-openai     | `sashabaranov/go-openai`         | OpenAI chat completions + streaming |
| go-github     | `google/go-github/v62`           | GitHub REST API — PR and check fetching |
| Cobra         | `spf13/cobra`                    | CLI flags: `--since`, `--model`, `--plain` |
| yaml.v3       | `gopkg.in/yaml.v3`               | Config file parse/write |

### go.mod

```go
module github.com/yourusername/standup

go 1.22

require (
    github.com/charmbracelet/bubbles  v0.18.0
    github.com/charmbracelet/bubbletea v0.26.4
    github.com/charmbracelet/lipgloss  v0.11.0
    github.com/google/go-github/v62    v62.0.0
    github.com/sashabaranov/go-openai  v1.24.0
    github.com/spf13/cobra             v1.8.1
    golang.org/x/oauth2                v0.20.0
    gopkg.in/yaml.v3                   v3.0.1
)
```

---

## 7. OpenAI Prompt Strategy

The AI layer builds a structured prompt from all available data before calling the API. The prompt is designed to produce brief, professional, and natural-sounding standup notes — not a log dump.

### System prompt

```
You are a helpful assistant that writes concise daily standup notes
for software engineers. Be brief, professional, and clear.
Format the output in exactly three sections: Yesterday, Today, Blockers.
Keep each section to 2-4 sentences maximum.
Do not include commit hashes or PR numbers unless directly relevant.
Write in first person, past tense for Yesterday, present/future for Today.
```

### User prompt (constructed at runtime)

```
Here is my activity from the last 12 hours:

COMMITS:
- [api-service] feat: add JWT refresh token rotation (a3f9c2b)
- [api-service] fix: null pointer in middleware handler (91ec44d)
- [infra] chore: bump Go version to 1.22 (d7a1003)

PULL REQUESTS:
- #421 feat: oauth2 refresh — APPROVED — CI pass — 2h ago
- #418 fix: parser panic — IN REVIEW — CI pass — 4h ago

TODAY (my notes): Continue oauth2 work, review #418
BLOCKERS: none

Write a natural standup summary.
```

### Streaming

Use the OpenAI streaming API (`CreateChatCompletionStream`) so the response appears token-by-token in the TUI. Send each token chunk as a Bubbletea `tea.Msg` to update the summary view in real time.

> **Cost note:** `gpt-4o-mini` at ~400 tokens per standup costs approximately $0.0002 per run — essentially free. `gpt-4o` costs ~10x more but produces noticeably better prose. Let users choose via config.

---

## 8. GitHub API Details

Only the REST API is needed — no GraphQL. Use the `google/go-github` client with a Personal Access Token (`repo` scope is sufficient).

### What to fetch

- PRs created or updated in the last N hours (matching the `since` window)
- For each PR: number, title, author, state (`open`/`merged`/`closed`), review state (`approved`/`changes_requested`/`pending`), CI check status (`pass`/`fail`/`pending`), `created_at`
- Endpoint: `GET /repos/{owner}/{repo}/pulls?state=all&sort=updated&per_page=50`
- Filter client-side by `updated_at >= time.Now().Add(-since)`

### Required token scopes

- `repo` — to read pull requests and check runs
- No write scopes needed

> **Setup tip:** Generate a fine-grained PAT at `github.com/settings/tokens` with read-only access to the specific repos you configure. Store it in `config.yaml` or export as `GITHUB_TOKEN`.

---

## 9. Recommended Build Order

Build bottom-up: stable dependencies first, TUI last. Each step is independently testable.

| Step | File(s) | Goal |
|------|---------|------|
| 1  | `go.mod` + `cmd/standup/main.go`      | Project scaffold, cobra CLI, `standup init` command |
| 2  | `internal/config/config.go`           | Load/save YAML, expand `~`, env var overrides, defaults |
| 3  | `internal/git/scanner.go`             | Walk `scan_dirs`, run `git log --since`, parse commit output |
| 4  | `internal/github/client.go`           | REST client, auth, fetch PRs, filter by time window |
| 5  | `internal/ai/summarize.go`            | Build prompt, call OpenAI streaming, send tokens as `tea.Msg` |
| 6  | `internal/formatter/format.go`        | Plain, Markdown, Slack mrkdwn formatters |
| 7  | `internal/history/history.go`         | Save/load `history.json`, 30-entry cap |
| 8  | `internal/share/share.go`             | Clipboard (OS-aware), Slack webhook POST |
| 9  | `internal/ui/styles.go`               | Lipgloss tokens — colors, borders, shared styles |
| 10 | `internal/ui/main_view.go`            | Main standup screen — commits, PRs, edit fields |
| 11 | `internal/ui/summary_view.go`         | Streaming AI summary view with live token updates |
| 12 | `internal/ui/history_view.go`         | Two-pane history browser |
| 13 | `internal/ui/config_view.go`          | Inline config editor |
| 14 | `internal/ui/model.go`                | Root model, tab routing, wire all views together |
| 15 | `README.md` + `config.example.yaml`   | Documentation, demo GIF for LinkedIn post |

---

## 10. Keyboard Shortcuts

| Key              | Context         | Action |
|------------------|-----------------|--------|
| `1` / `2` / `3` / `4` | any tab   | Switch to that tab |
| `e`              | main, summary   | Enter edit mode for Today / Blockers |
| `enter`          | editing         | Add new item / new line |
| `tab`            | editing         | Switch between Today and Blockers fields |
| `esc`            | editing         | Exit edit mode, save content |
| `s`              | main            | Generate AI summary (calls OpenAI) |
| `s`              | summary         | Post summary to Slack |
| `c`              | any tab         | Copy current content to clipboard |
| `r`              | main            | Refresh — re-scan git + re-fetch GitHub PRs |
| `r`              | summary         | Regenerate — call OpenAI again for a new summary |
| `↑` / `↓` or `j`/`k` | history   | Navigate past standup entries |
| `q` / `ctrl+c`  | any tab         | Quit (saves today's standup to history first) |

---

## 11. CLI Commands & Flags

```bash
# Open the interactive TUI (default)
standup

# Create config file at ~/.standup/config.yaml
standup init

# Print standup to stdout, no TUI (pipe-friendly)
standup show

# Custom time window
standup --since 24h

# Filter commits by author
standup --author "Alice"

# Choose AI model for this run
standup --model gpt-4o-mini

# Skip AI, just show raw commits/PRs
standup --no-ai
```

---

## 12. Implementation Notes for Claude Code

### git/scanner.go

```go
// Key command to run for each repo:
cmd := exec.Command("git", "log",
    "--format=%H|%s|%an|%ai",
    "--since="+cfg.Since,
    "--author="+cfg.Author,
)
cmd.Dir = repoPath
```

- Walk each `scan_dirs` entry one level deep looking for `.git` directories
- Parse each line: `hash|subject|author|date`
- Group results by repo name (`filepath.Base(repoPath)`)

### github/client.go

```go
// Auth setup
ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Github.Token})
tc := oauth2.NewClient(ctx, ts)
client := github.NewClient(tc)

// Fetch PRs updated in the last window
opts := &github.PullRequestListOptions{
    State:     "all",
    Sort:      "updated",
    Direction: "desc",
    ListOptions: github.ListOptions{PerPage: 50},
}
prs, _, err := client.PullRequests.List(ctx, owner, repo, opts)
// Then filter: pr.UpdatedAt.After(time.Now().Add(-since))
```

### ai/summarize.go

```go
// Stream tokens back as tea.Msg
stream, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
    Model:     cfg.OpenAI.Model,
    MaxTokens: cfg.OpenAI.MaxTokens,
    Messages:  buildMessages(data),
    Stream:    true,
})
for {
    resp, err := stream.Recv()
    if err == io.EOF { break }
    token := resp.Choices[0].Delta.Content
    program.Send(TokenMsg{Token: token})
}
```

### ui/model.go — Bubbletea structure

```go
type Model struct {
    cfg         *config.Config
    width       int
    height      int
    activeTab   tab           // tabMain | tabSummary | tabHistory | tabConfig
    mainView    MainModel
    summaryView SummaryModel
    historyView HistoryModel
    configView  ConfigModel
    statusMsg   string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle global keys (tab switching, quit)
    // Delegate all other keys to active tab's Update()
}
```

### ui/summary_view.go — live streaming

```go
type TokenMsg struct{ Token string }

// In Update():
case TokenMsg:
    m.summary += msg.Token
    return m, listenForTokens(m.tokenChan)  // recursive cmd

// In View():
// Render m.summary as it grows
// Show spinner if m.streaming == true
```

---

## 13. LinkedIn Post Tips

A great demo video will make this project stand out. Here's what to record:

1. Start with terminal open — run `standup`
2. Show commits and PRs auto-populated (no typing)
3. Type one line in the Today field
4. Press `s` — watch the AI summary stream in live
5. Press `c` — "copied to clipboard" confirmation
6. Paste into Slack in the same video

**Suggested caption:**

> "Built a Go TUI that writes my daily standup from git commits + GitHub PRs, then uses GPT-4o to generate a clean summary — one keypress, done. Built with Bubbletea + Lipgloss. Open source, link in comments."

> **Pro tip:** Record the demo at 1.5x speed and export as a GIF using `asciinema` + `agg`. A GIF auto-plays on LinkedIn and gets 3–5x more impressions than a static screenshot.