package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gitscan "github.com/rifat977/standup/internal/git"
	ghclient "github.com/rifat977/standup/internal/github"
)

type editTarget int

const (
	editNone editTarget = iota
	editToday
	editBlocker
)

type MainModel struct {
	commits []gitscan.Commit
	prs     []ghclient.PR

	today   textarea.Model
	blocker textarea.Model

	editing  editTarget
	width    int
	height   int
	loading  bool
	loadInfo string
}

func newMainModel() MainModel {
	t := textarea.New()
	t.Placeholder = "what you'll work on today..."
	t.SetWidth(60)
	t.SetHeight(2)
	t.ShowLineNumbers = false

	b := textarea.New()
	b.Placeholder = "anything blocking you?"
	b.SetWidth(60)
	b.SetHeight(2)
	b.ShowLineNumbers = false

	return MainModel{today: t, blocker: b, loading: true, loadInfo: "scanning repos..."}
}

// dataLoadedMsg is fired by the root once git+github fetches return.
type dataLoadedMsg struct {
	Commits []gitscan.Commit
	PRs     []ghclient.PR
}

func (m *MainModel) setSize(w, h int) {
	m.width, m.height = w, h
	tw := w - 8
	if tw < 20 {
		tw = 20
	}
	m.today.SetWidth(tw)
	m.blocker.SetWidth(tw)
}

func (m MainModel) Today() string   { return strings.TrimSpace(m.today.Value()) }
func (m MainModel) Blocker() string { return strings.TrimSpace(m.blocker.Value()) }

func (m MainModel) Update(msg tea.Msg) (MainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case dataLoadedMsg:
		m.commits = msg.Commits
		m.prs = msg.PRs
		m.loading = false
		return m, nil
	case tea.KeyMsg:
		if m.editing != editNone {
			switch msg.String() {
			case "esc":
				m.exitEdit()
				return m, nil
			case "tab":
				m.toggleEdit()
				return m, nil
			}
			var cmd tea.Cmd
			if m.editing == editToday {
				m.today, cmd = m.today.Update(msg)
			} else {
				m.blocker, cmd = m.blocker.Update(msg)
			}
			return m, cmd
		}
		switch msg.String() {
		case "e":
			m.enterEdit(editToday)
			return m, textarea.Blink
		case "b":
			m.enterEdit(editBlocker)
			return m, textarea.Blink
		}
	}
	return m, nil
}

func (m *MainModel) enterEdit(t editTarget) {
	m.editing = t
	if t == editToday {
		m.today.Focus()
		m.blocker.Blur()
	} else {
		m.blocker.Focus()
		m.today.Blur()
	}
}

func (m *MainModel) toggleEdit() {
	if m.editing == editToday {
		m.enterEdit(editBlocker)
	} else {
		m.enterEdit(editToday)
	}
}

func (m *MainModel) exitEdit() {
	m.editing = editNone
	m.today.Blur()
	m.blocker.Blur()
}

// Editing reports whether a text field has focus (root uses this to gate global keys).
func (m MainModel) Editing() bool { return m.editing != editNone }

func (m MainModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Commits & PRs"))
	b.WriteString(mutedStyle.Render("  (last activity)\n\n"))

	if m.loading {
		b.WriteString(mutedStyle.Render("  " + m.loadInfo + "\n"))
	} else if len(m.commits) == 0 && len(m.prs) == 0 {
		b.WriteString(mutedStyle.Render("  (no commits or PRs in window)\n"))
	} else {
		groups, order := gitscan.GroupByRepo(m.commits)
		for _, repo := range order {
			b.WriteString("  " + repoHeaderStyle.Render("── "+repo+" ──") + "\n")
			for _, c := range groups[repo] {
				fmt.Fprintf(&b, "    %s  %s\n",
					mutedStyle.Render(c.Hash), primary.Render(c.Subject))
			}
		}
		if len(m.prs) > 0 {
			b.WriteString("\n  " + repoHeaderStyle.Render("Pull Requests") + "\n")
			for _, p := range m.prs {
				ci := ciStyle(p.CI).Render("CI " + p.CI)
				rev := reviewStyle(p.Review).Render(strings.ToUpper(p.Review))
				fmt.Fprintf(&b, "    #%d  %s  %s  %s  %s\n",
					p.Number, primary.Render(truncate(p.Title, 50)),
					rev, ci, mutedStyle.Render(p.AgeString()))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(sectionLabel("Today", m.editing == editToday))
	b.WriteString("\n")
	b.WriteString(renderField(m.today, m.editing == editToday))
	b.WriteString("\n\n")
	b.WriteString(sectionLabel("Blockers", m.editing == editBlocker))
	b.WriteString("\n")
	b.WriteString(renderField(m.blocker, m.editing == editBlocker))
	return b.String()
}

func sectionLabel(title string, active bool) string {
	if active {
		return titleStyle.Render("● " + title)
	}
	return repoHeaderStyle.Render(title)
}

func renderField(t textarea.Model, active bool) string {
	style := lipgloss.NewStyle().Padding(0, 1)
	if active {
		return editingFrame.Render(t.View())
	}
	v := strings.TrimSpace(t.Value())
	if v == "" {
		v = mutedStyle.Render("(press " + tabHint(t) + " to edit)")
	}
	return style.Render(v)
}

func tabHint(t textarea.Model) string {
	if t.Placeholder != "" && strings.Contains(t.Placeholder, "blocking") {
		return "b"
	}
	return "e"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
