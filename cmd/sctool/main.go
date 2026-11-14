package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	accent      = lipgloss.Color("#7D56F4")
	muted       = lipgloss.Color("#6C6C6C")
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(accent).MarginTop(1)
	subtle      = lipgloss.NewStyle().Foreground(muted)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)
	badge       = lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(lipgloss.Color("#1A1A1A"))
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
		m.view.Width, m.view.Height = msg.Width, max(1, msg.Height-2)
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
	mode := badge.Background(accent).Render("NORMAL")
	if m.inserting {
		mode = badge.Background(lipgloss.Color("#04B575")).Render("INSERT")
	}
	status := subtle.Render("  c commodity · l location · j/k gg/G ^d/^u scroll · ? help · q quit")
	if !m.data.FetchedAt.IsZero() {
		status = subtle.Render("  data "+m.data.FetchedAt.Format(time.TimeOnly)) + status
	}
	if m.err != nil {
		status = "  " + errorStyle.Render(m.err.Error())
	}
	prompt := ""
	if m.inserting {
		prompt = subtle.Render(string(m.kind)+" ") + m.input.View()
	}
	return m.view.View() + "\n" + mode + status + "\n" + prompt
}

func help() string {
	return titleStyle.Render("sctool — Star Citizen commodity prices") + "\n" + subtle.Render(strings.Join([]string{
		"",
		"  c      search a commodity (fuzzy, e.g. 'quant')",
		"  l      search a location (fuzzy, e.g. 'new babbage')",
		"  /      search again in the current mode",
		"  enter  run search   esc  cancel",
		"  j/k    scroll       ctrl-d/ctrl-u  half page   gg/G  top/bottom",
		"  q      quit",
		"",
		"  Prices are aUEC per SCU from api.uexcorp.space, refreshed at most once a minute.",
	}, "\n"))
}

func render(data snapshot, kind searchKind, query string) string {
	var sections []string
	if kind == commoditySearch {
		identity := func(name string) []string { return []string{name} }
		for _, name := range topMatches(query, commodityNames(data.Prices), identity, 3) {
			sections = append(sections, renderCommodity(data.Prices, name))
		}
	} else {
		keys := func(t terminal) []string { return []string{t.Name, t.DisplayName, t.Nickname} }
		for _, t := range topMatches(query, tradingTerminals(data), keys, 3) {
			sections = append(sections, renderLocation(data.Prices, t))
		}
	}
	if len(sections) == 0 {
		return titleStyle.Render(fmt.Sprintf("No %s matches %q", kind, query))
	}
	return strings.Join(sections, "\n")
}

func renderCommodity(prices []price, name string) string {
	var rows [][]string
	for _, line := range commodityReport(prices, name) {
		rows = append(rows, []string{line.System, purchase(line.Buy, true), sale(line.Sell, false)})
	}
	return titleStyle.Render("◆ "+name) + "\n" + grid([]string{"SYSTEM", "CHEAPEST BUY", "BEST SELL"}, rows)
}

func renderLocation(prices []price, t terminal) string {
	title := titleStyle.Render("◆ "+t.Name) + subtle.Render("  "+t.DisplayName+" · "+t.System)
	lines := locationReport(prices, t.ID)
	if len(lines) == 0 {
		return title + "\n" + subtle.Render("  No commodities for sale here.")
	}
	var rows [][]string
	for _, line := range lines {
		item := fmt.Sprintf("%s - %s/SCU - %s SCU", line.Item.Commodity, money(line.Item.Buy), money(line.Item.Stock))
		rows = append(rows, []string{item, sale(line.InSystem, false), sale(line.Anywhere, true)})
	}
	return title + "\n" + grid([]string{"COMMODITY", "BEST SELL IN SYSTEM", "BEST SELL ANYWHERE"}, rows)
}

func grid(headers []string, rows [][]string) string {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(subtle).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		}).
		Render()
}

func purchase(p *price, withStock bool) string {
	if p == nil {
		return "-"
	}
	text := fmt.Sprintf("%s - %s/SCU", p.Terminal, money(p.Buy))
	if withStock {
		text += fmt.Sprintf(" - %s SCU", money(p.Stock))
	}
	return text
}

func sale(p *price, withSystem bool) string {
	if p == nil {
		return "-"
	}
	where := p.Terminal
	if withSystem {
		where += " (" + p.System + ")"
	}
	return fmt.Sprintf("%s - %s/SCU", where, money(p.Sell))
}

func money(value float64) string {
	digits := strconv.FormatInt(int64(value), 10)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}

func main() {
	input := textinput.New()
	input.Placeholder = "type to search, enter to run"
	input.Prompt = "❯ "
	program := tea.NewProgram(model{kind: commoditySearch, input: input, view: viewport.New(0, 0)}, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
