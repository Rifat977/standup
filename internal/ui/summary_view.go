package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rifat977/standup/internal/ai"
)

type SummaryModel struct {
	summary   string
	streaming bool
	editing   bool
	err       error
	tokens    int
	spin      spinner.Model
	editor    textarea.Model
	model     string
	width     int
}

func newSummaryModel(modelName string) SummaryModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	t := textarea.New()
	t.SetHeight(10)
	t.SetWidth(60)
	t.ShowLineNumbers = false

	return SummaryModel{spin: s, editor: t, model: modelName}
}

func (m *SummaryModel) setSize(w, h int) {
	m.width = w
	tw := w - 8
	if tw < 20 {
		tw = 20
	}
	m.editor.SetWidth(tw)
	m.editor.SetHeight(maxInt(6, h-12))
}

// Editing reports whether the user is editing the summary text.
func (m SummaryModel) Editing() bool { return m.editing }

// Text returns the current summary content (edited if editing has happened).
func (m SummaryModel) Text() string {
	if m.editing || strings.TrimSpace(m.editor.Value()) != "" {
		return m.editor.Value()
	}
	return m.summary
}

func (m *SummaryModel) startStream() tea.Cmd {
	m.summary = ""
	m.tokens = 0
	m.err = nil
	m.streaming = true
	m.editing = false
	return m.spin.Tick
}

func (m SummaryModel) Update(msg tea.Msg) (SummaryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ai.TokenMsg:
		m.summary += msg.Token
		m.tokens++
		return m, nil
	case ai.DoneMsg:
		m.streaming = false
		m.err = msg.Err
		if m.err == nil {
			m.editor.SetValue(m.summary)
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		if m.streaming {
			return m, cmd
		}
		return m, nil
	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "esc":
				m.editing = false
				m.summary = m.editor.Value()
				return m, nil
			}
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "e":
			if !m.streaming {
				m.editing = true
				m.editor.SetValue(m.summary)
				m.editor.Focus()
				return m, textarea.Blink
			}
		}
	}
	return m, nil
}

func (m SummaryModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("AI Standup Summary"))
	b.WriteString(mutedStyle.Render("    model: " + m.model + "\n"))
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	if m.err != nil {
		b.WriteString(statusErr.Render("error: "+m.err.Error()) + "\n\n")
		b.WriteString(mutedStyle.Render("press r to retry\n"))
		return b.String()
	}

	if m.editing {
		b.WriteString(editingFrame.Render(m.editor.View()))
		b.WriteString("\n" + mutedStyle.Render("editing — esc to save"))
		return b.String()
	}

	if strings.TrimSpace(m.summary) == "" && m.streaming {
		b.WriteString(mutedStyle.Render(m.spin.View() + " contacting OpenAI...\n"))
		return b.String()
	}
	if strings.TrimSpace(m.summary) == "" {
		b.WriteString(mutedStyle.Render("press s on the main tab to generate a summary\n"))
		return b.String()
	}

	b.WriteString(primary.Render(m.summary))
	b.WriteString("\n\n")
	if m.streaming {
		b.WriteString(mutedStyle.Render(m.spin.View() + " streaming..."))
	} else {
		b.WriteString(mutedStyle.Render("done."))
	}
	b.WriteString(mutedStyle.Render("        tokens: ") + primary.Render(itoa(m.tokens)))
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
