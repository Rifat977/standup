package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rifat977/standup/internal/ai"
	"github.com/rifat977/standup/internal/config"
	gitscan "github.com/rifat977/standup/internal/git"
	ghclient "github.com/rifat977/standup/internal/github"
	"github.com/rifat977/standup/internal/history"
	"github.com/rifat977/standup/internal/share"
)

type tab int

const (
	tabMain tab = iota
	tabSummary
	tabHistory
	tabConfig
)

// Model is the root bubbletea model.
type Model struct {
	cfg       *config.Config
	width     int
	height    int
	active    tab
	main      MainModel
	summary   SummaryModel
	history   HistoryModel
	config    ConfigModel
	status    string
	statusExp time.Time
	prog      *tea.Program
	tokenCh   chan any
}

// New constructs the root model with default state.
func New(cfg *config.Config) *Model {
	return &Model{
		cfg:     cfg,
		main:    newMainModel(),
		summary: newSummaryModel(cfg.OpenAI.Model),
		history: newHistoryModel(),
		config:  newConfigModel(cfg),
	}
}

// Bind hands the running tea.Program to the model so it can Send streamed messages.
func (m *Model) Bind(p *tea.Program) { m.prog = p }

func (m *Model) Init() tea.Cmd {
	return tea.Batch(loadDataCmd(m.cfg), tea.WindowSize())
}

// loadDataCmd runs git+github fetch in a goroutine and returns dataLoadedMsg.
func loadDataCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		commits, _ := gitscan.Collect(cfg)
		prs, _ := ghclient.Collect(context.Background(), cfg)
		return dataLoadedMsg{Commits: commits, PRs: prs}
	}
}

type statusMsg struct{ text string }
type clearStatusMsg struct{}

func (m *Model) flash(s string) tea.Cmd {
	m.status = s
	m.statusExp = time.Now().Add(2 * time.Second)
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.main.setSize(msg.Width, msg.Height)
		m.summary.setSize(msg.Width, msg.Height)
		m.history.setSize(msg.Width, msg.Height)
		m.config.setSize(msg.Width, msg.Height)
		return m, nil

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case dataLoadedMsg:
		var cmd tea.Cmd
		m.main, cmd = m.main.Update(msg)
		return m, cmd

	case ai.TokenMsg, ai.DoneMsg:
		var cmd tea.Cmd
		m.summary, cmd = m.summary.Update(msg)
		next := m.listenStream()
		if next != nil {
			cmd = tea.Batch(cmd, next)
		}
		if _, isDone := msg.(ai.DoneMsg); isDone {
			cmd = tea.Batch(cmd, m.flash("summary ready"))
		}
		return m, cmd

	case tea.KeyMsg:
		// Handle global keys only when no field is being edited.
		editing := (m.active == tabMain && m.main.Editing()) ||
			(m.active == tabSummary && m.summary.Editing()) ||
			(m.active == tabConfig && m.config.Editing())

		if !editing {
			switch msg.String() {
			case "1":
				m.active = tabMain
				return m, nil
			case "2":
				m.active = tabSummary
				return m, nil
			case "3":
				m.active = tabHistory
				m.history.reload()
				return m, nil
			case "4":
				m.active = tabConfig
				return m, nil
			case "q", "ctrl+c":
				m.saveHistoryOnExit()
				return m, tea.Quit
			}
		}

		// Tab-specific actions handled before delegating.
		switch m.active {
		case tabMain:
			if !m.main.Editing() {
				switch msg.String() {
				case "s":
					return m, m.startStream()
				case "r":
					return m, loadDataCmd(m.cfg)
				case "c":
					return m, m.copyMain()
				}
			}
		case tabSummary:
			if !m.summary.Editing() {
				switch msg.String() {
				case "r":
					return m, m.startStream()
				case "c":
					return m, m.copySummary()
				case "s":
					return m, m.postSlack()
				}
			}
		case tabHistory:
			switch msg.String() {
			case "c":
				return m, m.copyHistory()
			}
		}

		// Delegate to active sub-model.
		var cmd tea.Cmd
		switch m.active {
		case tabMain:
			m.main, cmd = m.main.Update(msg)
		case tabSummary:
			m.summary, cmd = m.summary.Update(msg)
		case tabHistory:
			m.history, cmd = m.history.Update(msg)
		case tabConfig:
			m.config, cmd = m.config.Update(msg)
		}
		return m, cmd
	}

	// Non-key messages (spinner ticks etc) — broadcast to active model.
	var cmd tea.Cmd
	switch m.active {
	case tabMain:
		m.main, cmd = m.main.Update(msg)
	case tabSummary:
		m.summary, cmd = m.summary.Update(msg)
	case tabHistory:
		m.history, cmd = m.history.Update(msg)
	case tabConfig:
		m.config, cmd = m.config.Update(msg)
	}
	return m, cmd
}

func (m *Model) startStream() tea.Cmd {
	if m.prog == nil {
		return m.flash("internal: program not bound")
	}
	d := ai.Data{
		Commits: m.main.commits,
		PRs:     m.main.prs,
		Today:   m.main.Today(),
		Blocker: m.main.Blocker(),
	}
	m.tokenCh = make(chan any, 64)
	cmd := m.summary.startStream()
	m.active = tabSummary
	go ai.Stream(context.Background(), m.cfg, d, m.tokenCh)
	return tea.Batch(cmd, m.listenStream())
}

func (m *Model) listenStream() tea.Cmd {
	if m.tokenCh == nil {
		return nil
	}
	ch := m.tokenCh
	return func() tea.Msg {
		v, ok := <-ch
		if !ok {
			return ai.DoneMsg{}
		}
		return v
	}
}

func (m *Model) copyMain() tea.Cmd {
	d := ai.Data{
		Commits: m.main.commits, PRs: m.main.prs,
		Today: m.main.Today(), Blocker: m.main.Blocker(),
	}
	if err := share.Copy(ai.BuildUserPrompt(d)); err != nil {
		return m.flash("copy failed: " + err.Error())
	}
	return m.flash("copied raw activity")
}

func (m *Model) copySummary() tea.Cmd {
	if strings.TrimSpace(m.summary.Text()) == "" {
		return m.flash("nothing to copy yet")
	}
	if err := share.Copy(m.summary.Text()); err != nil {
		return m.flash("copy failed: " + err.Error())
	}
	return m.flash("copied summary")
}

func (m *Model) copyHistory() tea.Cmd {
	e, ok := m.history.Selected()
	if !ok {
		return m.flash("no entry selected")
	}
	if err := share.Copy(e.Summary); err != nil {
		return m.flash("copy failed: " + err.Error())
	}
	return m.flash("copied " + e.Date.Format("Jan 02"))
}

func (m *Model) postSlack() tea.Cmd {
	if strings.TrimSpace(m.summary.Text()) == "" {
		return m.flash("nothing to post yet")
	}
	if err := share.PostSlack(m.cfg, m.summary.Text()); err != nil {
		return m.flash("slack: " + err.Error())
	}
	return m.flash("posted to slack")
}

func (m *Model) saveHistoryOnExit() {
	s := strings.TrimSpace(m.summary.Text())
	if s == "" {
		return
	}
	_ = history.Save(history.Entry{
		Date:    time.Now(),
		Summary: s,
		Today:   m.main.Today(),
		Blocker: m.main.Blocker(),
	})
}

func (m *Model) View() string {
	header := m.renderHeader()
	var body string
	switch m.active {
	case tabMain:
		body = m.main.View()
	case tabSummary:
		body = m.summary.View()
	case tabHistory:
		body = m.history.View()
	case tabConfig:
		body = m.config.View()
	}
	footer := m.renderFooter()
	return fmt.Sprintf("%s\n\n%s\n\n%s", header, body, footer)
}

func (m *Model) renderHeader() string {
	tabs := []struct {
		id    tab
		label string
	}{
		{tabMain, "[1] main"},
		{tabSummary, "[2] summary"},
		{tabHistory, "[3] history"},
		{tabConfig, "[4] config"},
	}
	var parts []string
	for _, t := range tabs {
		if t.id == m.active {
			parts = append(parts, tabActiveStyle.Render(t.label))
		} else {
			parts = append(parts, tabStyle.Render(t.label))
		}
	}
	title := titleStyle.Render("standup")
	right := mutedStyle.Render(time.Now().Format("Mon Jan 02"))
	return title + "  " + strings.Join(parts, " ") + "    " + right
}

func (m *Model) renderFooter() string {
	var hints string
	switch m.active {
	case tabMain:
		hints = "e:today  b:blocker  s:summary  r:refresh  c:copy  1-4:tabs  q:quit"
	case tabSummary:
		hints = "e:edit  c:copy  s:slack  r:regenerate  1-4:tabs  q:quit"
	case tabHistory:
		hints = "↑↓:navigate  c:copy  1-4:tabs  q:quit"
	case tabConfig:
		hints = "↑↓:nav  enter:edit  s:save  1-4:tabs  q:quit"
	}
	if m.status != "" {
		hints = primary.Render(m.status) + "    " + mutedStyle.Render(hints)
	} else {
		hints = mutedStyle.Render(hints)
	}
	return footerStyle.Render(hints)
}
