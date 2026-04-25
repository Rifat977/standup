package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	b.WriteString(renderStandupBody(m.summary, m.width))
	b.WriteString("\n\n")
	if m.streaming {
		b.WriteString(mutedStyle.Render(m.spin.View() + " streaming..."))
	} else {
		b.WriteString(mutedStyle.Render("done."))
	}
	b.WriteString(mutedStyle.Render("        tokens: ") + primary.Render(itoa(m.tokens)))
	return b.String()
}

// Section styles — lipgloss handles wrapping (via Width) and indentation
// (via PaddingLeft / MarginLeft), so we never hand-pad strings.
var (
	sectionWidth = func(viewWidth int) int {
		w := viewWidth - 4
		if w < 40 {
			return 60
		}
		return w
	}

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginTop(1)

	bodyStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			PaddingLeft(3)

	bulletStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			PaddingLeft(3)

	bulletMarker = lipgloss.NewStyle().
			Foreground(colorAccent).
			SetString("•")
)

type section struct {
	title string
	body  []string // freeform paragraphs
	bul   []string // bullet items
}

// renderStandupBody composes the AI output into vertically-joined lipgloss
// blocks: a colored header per section, then either a wrapped paragraph block
// or a bulleted list. lipgloss does the wrapping via Width().
func renderStandupBody(s string, viewWidth int) string {
	w := sectionWidth(viewWidth)
	sections := parseSections(s)

	icons := map[string]string{
		"Yesterday": "✓",
		"Today":     "▸",
		"Blockers":  "⚠",
	}

	var blocks []string
	for _, sec := range sections {
		if sec.title != "" {
			icon := icons[sec.title]
			if icon == "" {
				icon = "•"
			}
			blocks = append(blocks, headerStyle.Render(icon+"  "+sec.title))
		}
		if len(sec.body) > 0 {
			text := strings.Join(sec.body, " ")
			blocks = append(blocks, bodyStyle.Width(w).Render(text))
		}
		for _, item := range sec.bul {
			marker := bulletMarker.String() + " "
			blocks = append(blocks, bulletStyle.Width(w).Render(marker+item))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

// parseSections groups lines under their nearest section heading. Bullet
// lines are kept separate so they can be styled individually.
func parseSections(s string) []section {
	var out []section
	cur := -1
	flush := func() { /* no-op; kept for symmetry */ }

	for _, raw := range strings.Split(s, "\n") {
		line := strings.TrimSpace(stripBold(raw))
		if line == "" {
			continue
		}
		if title, ok := matchHeader(line); ok {
			out = append(out, section{title: title})
			cur = len(out) - 1
			continue
		}
		if cur < 0 {
			out = append(out, section{})
			cur = 0
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			out[cur].bul = append(out[cur].bul, strings.TrimSpace(line[2:]))
		} else {
			out[cur].body = append(out[cur].body, line)
		}
	}
	flush()
	return out
}

// matchHeader returns (canonicalTitle, true) for a recognised section header.
// Accepts variants like "Yesterday", "Yesterday:", "## Yesterday".
func matchHeader(line string) (string, bool) {
	clean := stripBold(line)
	clean = strings.TrimPrefix(clean, "## ")
	clean = strings.TrimPrefix(clean, "# ")
	clean = strings.TrimSuffix(clean, ":")
	clean = strings.TrimSpace(clean)
	switch strings.ToLower(clean) {
	case "yesterday":
		return "Yesterday", true
	case "today":
		return "Today", true
	case "blockers", "blocker":
		return "Blockers", true
	}
	return "", false
}

func stripBold(s string) string { return strings.ReplaceAll(s, "**", "") }

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
