package main

import (
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func press(m model, keys ...string) model {
	for _, key := range keys {
		message := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if key == "enter" {
			message = tea.KeyMsg{Type: tea.KeyEnter}
		}
		next, _ := m.Update(message)
		m = next.(model)
	}
	return m
}

func TestTopCursor(t *testing.T) {
	routes := make([]route, 30)
	for index := range routes {
		routes[index] = route{Buy: price{Commodity: string(rune('A' + index)), Buy: 1}, Sell: price{Sell: 2}}
	}
	m := model{showing: topView, routes: routes, view: viewport.New(80, 10)}
	m.moveCursor(0)

	m = press(m, "k")
	if m.cursor != 0 {
		t.Fatalf("k at top moved cursor to %d", m.cursor)
	}
	m = press(m, "G")
	if m.cursor != 29 {
		t.Fatalf("G moved cursor to %d", m.cursor)
	}
	if line := m.cursor + topHeaderLines; line < m.view.YOffset || line >= m.view.YOffset+m.view.Height {
		t.Fatalf("selected line %d not visible in offset %d height %d", line, m.view.YOffset, m.view.Height)
	}
	m = press(m, "j", "g", "g")
	if m.cursor != 0 || m.view.YOffset != 0 {
		t.Fatalf("gg = cursor %d offset %d", m.cursor, m.view.YOffset)
	}
	m = press(m, "j", "j", "enter")
	if m.kind != commoditySearch || m.routes[m.cursor].Buy.Commodity != "C" {
		t.Fatalf("enter = kind %q on %q", m.kind, m.routes[m.cursor].Buy.Commodity)
	}
}
