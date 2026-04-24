package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rifat977/standup/internal/config"
)

type configField struct {
	label string
	get   func(*config.Config) string
	set   func(*config.Config, string)
	mask  bool
}

type ConfigModel struct {
	cfg      *config.Config
	fields   []configField
	inputs   []textinput.Model
	selected int
	editing  bool
	saved    bool
	saveErr  error
	width    int
}

func newConfigModel(cfg *config.Config) ConfigModel {
	fields := []configField{
		{label: "Author", get: func(c *config.Config) string { return c.Author }, set: func(c *config.Config, v string) { c.Author = v }},
		{label: "Since (e.g. 12h)", get: func(c *config.Config) string { return c.Since }, set: func(c *config.Config, v string) { c.Since = v }},
		{label: "Scan dirs (comma-separated)",
			get: func(c *config.Config) string { return strings.Join(c.ScanDirs, ", ") },
			set: func(c *config.Config, v string) { c.ScanDirs = splitCSV(v) }},
		{label: "GitHub token", mask: true,
			get: func(c *config.Config) string { return c.GitHub.Token },
			set: func(c *config.Config, v string) { c.GitHub.Token = v }},
		{label: "GitHub repos (owner/repo, comma-separated)",
			get: func(c *config.Config) string { return strings.Join(c.GitHub.Repos, ", ") },
			set: func(c *config.Config, v string) { c.GitHub.Repos = splitCSV(v) }},
		{label: "OpenAI API key", mask: true,
			get: func(c *config.Config) string { return c.OpenAI.APIKey },
			set: func(c *config.Config, v string) { c.OpenAI.APIKey = v }},
		{label: "OpenAI model",
			get: func(c *config.Config) string { return c.OpenAI.Model },
			set: func(c *config.Config, v string) { c.OpenAI.Model = v }},
		{label: "Slack webhook",
			get: func(c *config.Config) string { return c.Slack.WebhookURL },
			set: func(c *config.Config, v string) { c.Slack.WebhookURL = v }},
		{label: "Slack channel",
			get: func(c *config.Config) string { return c.Slack.Channel },
			set: func(c *config.Config, v string) { c.Slack.Channel = v }},
		{label: "Format (markdown|plain|slack)",
			get: func(c *config.Config) string { return c.Format },
			set: func(c *config.Config, v string) { c.Format = v }},
	}

	inputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		t := textinput.New()
		t.Width = 60
		t.SetValue(f.get(cfg))
		if f.mask {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}
		inputs[i] = t
	}
	return ConfigModel{cfg: cfg, fields: fields, inputs: inputs}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (m *ConfigModel) setSize(w, h int) { m.width = w }

// Editing reports whether a config field has focus.
func (m ConfigModel) Editing() bool { return m.editing }

func (m ConfigModel) Update(msg tea.Msg) (ConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "esc", "enter":
				m.fields[m.selected].set(m.cfg, m.inputs[m.selected].Value())
				m.inputs[m.selected].Blur()
				m.editing = false
				m.saved = false
				return m, nil
			}
			var cmd tea.Cmd
			m.inputs[m.selected], cmd = m.inputs[m.selected].Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.fields)-1 {
				m.selected++
			}
		case "enter", "e":
			m.editing = true
			m.inputs[m.selected].Focus()
			return m, textinput.Blink
		case "s":
			m.saveErr = m.cfg.Save()
			m.saved = m.saveErr == nil
		}
	}
	return m, nil
}

func (m ConfigModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Configuration"))
	b.WriteString(mutedStyle.Render("   ↑↓ navigate · enter edit · s save\n\n"))
	for i, f := range m.fields {
		marker := "  "
		if i == m.selected {
			marker = primary.Render("▸ ")
		}
		label := f.label
		if i == m.selected {
			label = tabActiveStyle.Render(label)
		} else {
			label = mutedStyle.Render(label)
		}
		fmt.Fprintf(&b, "%s%s\n", marker, label)
		if i == m.selected && m.editing {
			b.WriteString("    " + m.inputs[i].View() + "\n")
		} else {
			val := m.inputs[i].Value()
			if f.mask && val != "" {
				val = maskValue(val)
			}
			if val == "" {
				val = mutedStyle.Render("(empty)")
			}
			b.WriteString("    " + primary.Render(val) + "\n")
		}
	}
	b.WriteString("\n")
	if m.saveErr != nil {
		b.WriteString(statusErr.Render("save error: " + m.saveErr.Error()))
	} else if m.saved {
		b.WriteString(statusOK.Render("✓ saved"))
	}
	return b.String()
}

func maskValue(s string) string {
	if len(s) <= 6 {
		return strings.Repeat("•", len(s))
	}
	return strings.Repeat("•", len(s)-4) + s[len(s)-4:]
}
