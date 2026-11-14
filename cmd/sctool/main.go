package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type searchKind string

const (
	commoditySearch searchKind = "commodity"
	locationSearch  searchKind = "location"
)

type loadedMsg struct {
	data  snapshot
	err   error
	kind  searchKind
	query string
}

type model struct {
	inserting bool
	kind      searchKind
	input     textinput.Model
	view      viewport.Model
	data      snapshot
	err       error
	pendingG  bool
	ready     bool
}

func search(kind searchKind, query string) tea.Cmd {
	return func() tea.Msg {
		data, err := load()
		return loadedMsg{data, err, kind, query}
	}
}

func (m model) Init() tea.Cmd { return search("", "") }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.view.Width, m.view.Height = msg.Width, max(1, msg.Height-1)
		if !m.ready {
			m.view.SetContent(help())
			m.ready = true
		}
	case loadedMsg:
		if msg.data.Prices != nil {
			m.data = msg.data
		}
		m.err = msg.err
		if msg.query != "" {
			m.view.SetContent(render(m.data, msg.kind, msg.query))
			m.view.GotoTop()
		}
	case tea.KeyMsg:
		if m.inserting {
			return m.updateInsert(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m model) updateInsert(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.inserting = false
		m.input.Blur()
		return m, nil
	case "enter":
		m.inserting = false
		m.input.Blur()
		return m, search(m.kind, m.input.Value())
	case "ctrl+c":
		return m, tea.Quit
	}
	var command tea.Cmd
	m.input, command = m.input.Update(key)
	return m, command
}

func (m model) updateNormal(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	wasPendingG := m.pendingG
	m.pendingG = false
	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "c", "l", "/":
		if key.String() == "c" {
			m.kind = commoditySearch
		} else if key.String() == "l" {
			m.kind = locationSearch
		}
		m.inserting = true
		m.input.SetValue("")
		return m, m.input.Focus()
	case "j", "down":
		m.view.ScrollDown(1)
	case "k", "up":
		m.view.ScrollUp(1)
	case "ctrl+d":
		m.view.HalfPageDown()
	case "ctrl+u":
		m.view.HalfPageUp()
	case "G":
		m.view.GotoBottom()
	case "g":
		if wasPendingG {
			m.view.GotoTop()
		} else {
			m.pendingG = true
		}
	case "?":
		m.view.SetContent(help())
	}
	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "loading…"
	}
	// Reverse video takes the badge background from the theme color, foreground from the terminal background.
	badge := lipgloss.NewStyle().Reverse(true).Bold(true).Padding(0, 1)
	if m.inserting {
		return m.view.View() + "\n" + badge.Foreground(lipgloss.Color("2")).Render(string(m.kind)) + " " + m.input.View()
	}
	status := faint.Render("  c commodity  l location  ? help  q quit")
	if !m.data.FetchedAt.IsZero() {
		status = faint.Render("  "+m.data.FetchedAt.Format(time.TimeOnly)) + status
	}
	if m.err != nil {
		status = "  " + loss.Render(m.err.Error())
	}
	return m.view.View() + "\n" + badge.Foreground(lipgloss.Color("4")).Render("normal") + status
}

func help() string {
	keys := [][2]string{
		{"c", "search a commodity (fuzzy, e.g. quant)"},
		{"l", "search a location (fuzzy, e.g. new babbage)"},
		{"/", "search again in the current mode"},
		{"enter / esc", "run / cancel search"},
		{"j k", "scroll"},
		{"ctrl-d ctrl-u", "half page"},
		{"gg G", "top / bottom"},
		{"q", "quit"},
	}
	lines := []string{heading("sctool", "Star Citizen commodity prices · aUEC per SCU · api.uexcorp.space")}
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("  %s  %s", accent.Width(13).Render(key[0]), key[1]))
	}
	return strings.Join(lines, "\n")
}

func main() {
	input := textinput.New()
	input.Placeholder = "type to search, enter to run"
	input.Prompt = "❯ "
	input.PromptStyle = accent
	input.PlaceholderStyle = faint
	program := tea.NewProgram(model{kind: commoditySearch, input: input, view: viewport.New(0, 0)}, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
