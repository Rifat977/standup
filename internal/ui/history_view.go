package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rifat977/standup/internal/history"
)

type HistoryModel struct {
	entries  []history.Entry
	selected int
	width    int
	height   int
	err      error
}

func newHistoryModel() HistoryModel {
	entries, err := history.Load()
	return HistoryModel{entries: entries, err: err}
}

func (m *HistoryModel) setSize(w, h int) {
	m.width, m.height = w, h
}

func (m *HistoryModel) reload() {
	m.entries, m.err = history.Load()
	if m.selected >= len(m.entries) {
		m.selected = 0
	}
}

// Selected returns the currently highlighted entry, if any.
func (m HistoryModel) Selected() (history.Entry, bool) {
	if len(m.entries) == 0 {
		return history.Entry{}, false
	}
	return m.entries[m.selected], true
}

func (m HistoryModel) Update(msg tea.Msg) (HistoryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.entries)-1 {
				m.selected++
			}
		}
	}
	return m, nil
}

func (m HistoryModel) View() string {
	if m.err != nil {
		return statusErr.Render("error loading history: " + m.err.Error())
	}
	if len(m.entries) == 0 {
		return mutedStyle.Render("no history yet — quitting from a generated summary saves one.")
	}

	leftWidth := 22
	if m.width > 0 && m.width/3 < leftWidth {
		leftWidth = m.width / 3
	}

	var left strings.Builder
	left.WriteString(titleStyle.Render("Past standups") + "\n\n")
	for i, e := range m.entries {
		row := fmt.Sprintf("%s", e.Date.Format("Mon Jan 02"))
		if i == m.selected {
			row = "▸ " + tabActiveStyle.Render(row)
		} else {
			row = "  " + mutedStyle.Render(row)
		}
		left.WriteString(row + "\n")
	}

	var right strings.Builder
	if e, ok := m.Selected(); ok {
		right.WriteString(titleStyle.Render(e.Date.Format("Monday, Jan 02 2006")) + "\n\n")
		right.WriteString(primary.Render(strings.TrimSpace(e.Summary)))
	}

	leftBlock := lipgloss.NewStyle().Width(leftWidth).Render(left.String())
	rightBlock := lipgloss.NewStyle().Render(right.String())
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, "  ", rightBlock)
}
