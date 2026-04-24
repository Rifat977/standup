# standup

Terminal CLI that gathers your last 12 hours of git commits and GitHub PRs, lets you add today's plan and blockers, and uses OpenAI to generate a clean standup note — one keypress, done.

## Install

```bash
go install github.com/rifat977/standup/cmd/standup@latest
```

Or build from source:

```bash
git clone https://github.com/rifat977/standup
cd standup
go build -o standup ./cmd/standup
```

## Quickstart

```bash
standup init                 # writes ~/.standup/config.yaml
$EDITOR ~/.standup/config.yaml   # add GitHub token, OpenAI key, repos
standup                      # opens the TUI
```

Tabs: `1` main · `2` summary · `3` history · `4` config

On the main view: `e` edit *Today*, `b` edit *Blockers*, `s` generate AI summary, `r` refresh, `c` copy, `q` quit.

## Non-interactive

```bash
standup show --no-ai      # raw commit + PR dump
standup show              # generate via OpenAI, print to stdout
standup --plain           # plain text (good for piping)
standup --since 24h       # widen the window
```

## Config

Lives at `~/.standup/config.yaml`. Overrides via env: `OPENAI_API_KEY`, `GITHUB_TOKEN`, `SLACK_WEBHOOK_URL`.

See `config.example.yaml` for the full schema.

## Stack

Go · [bubbletea](https://github.com/charmbracelet/bubbletea) · [lipgloss](https://github.com/charmbracelet/lipgloss) · [go-openai](https://github.com/sashabaranov/go-openai) · [go-github](https://github.com/google/go-github)
