package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ANSI palette indices 0-15 so colors follow the terminal theme.
var (
	accent = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	bold   = lipgloss.NewStyle().Bold(true)
	faint  = lipgloss.NewStyle().Faint(true)
	cost   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	gain   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	loss   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	place  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

const (
	nameWidth = 22
	barWidth  = 10
	topCount  = 25
	none      = "—"
)

type column struct {
	header string
	right  bool
}

func render(data snapshot, kind searchKind, query string) string {
	var sections []string
	if kind == topView {
		return renderTop(data.Prices)
	}
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
		return heading(fmt.Sprintf("no %s matches %q", kind, query), "")
	}
	return strings.Join(sections, "\n\n")
}

func renderCommodity(prices []price, name string) string {
	lines := commodityReport(prices, name)
	margins := make([]float64, len(lines))
	for index, line := range lines {
		if line.Buy != nil && line.Sell != nil {
			margins[index] = line.Sell.Sell - line.Buy.Buy
		}
	}
	best := maxOf(margins)
	var rows [][]string
	for index, line := range lines {
		row := []string{bold.Render(line.System)}
		if line.Buy != nil {
			row = append(row, place.Render(truncate(line.Buy.Terminal, nameWidth)), cost.Render(money(line.Buy.Buy)), stock(line.Buy.Stock))
		} else {
			row = append(row, faint.Render(none), "", "")
		}
		if line.Sell != nil {
			row = append(row, place.Render(truncate(line.Sell.Terminal, nameWidth)), gain.Render(money(line.Sell.Sell)))
		} else {
			row = append(row, faint.Render(none), "")
		}
		if line.Buy != nil && line.Sell != nil {
			row = append(row, signed(margins[index]), bar(margins[index], best))
		} else {
			row = append(row, "", "")
		}
		rows = append(rows, row)
	}
	columns := []column{{"system", false}, {"buy at", false}, {"aUEC", true}, {"scu", true}, {"sell at", false}, {"aUEC", true}, {"margin", true}, {"", false}}
	return heading(name, fmt.Sprintf("%d systems", len(lines))) + "\n" + layout(columns, rows)
}

func renderLocation(prices []price, t terminal) string {
	title := heading(t.Name, strings.Join(nonEmpty(t.DisplayName, t.System), " · "))
	lines := locationReport(prices, t.ID)
	if len(lines) == 0 {
		return title + "\n" + faint.Render("  nothing for sale here")
	}
	margin := func(line locationLine) float64 {
		if line.Anywhere == nil {
			return math.Inf(-1)
		}
		return line.Anywhere.Sell - line.Item.Buy
	}
	slices.SortStableFunc(lines, func(a, b locationLine) int { return cmp.Compare(margin(b), margin(a)) })
	best := max(0, margin(lines[0]))
	var rows [][]string
	for _, line := range lines {
		row := []string{bold.Render(line.Item.Commodity), cost.Render(money(line.Item.Buy)), stock(line.Item.Stock)}
		row = append(row, target(line.InSystem, line.Item.Buy, false)...)
		row = append(row, target(line.Anywhere, line.Item.Buy, true)...)
		rows = append(rows, append(row, bar(margin(line), best)))
	}
	columns := []column{
		{"commodity", false}, {"aUEC", true}, {"scu", true},
		{"best in system", false}, {"aUEC", true}, {"margin", true},
		{"best anywhere", false}, {"aUEC", true}, {"margin", true}, {"", false},
	}
	return title + "\n" + layout(columns, rows)
}

func renderTop(prices []price) string {
	routes := topRoutes(prices, topCount)
	title := heading("Top margins", "cheapest in-stock buy → best sell anywhere · aUEC per SCU")
	if len(routes) == 0 {
		return title + "\n" + faint.Render("  no profitable routes in the current data")
	}
	roi := func(r route) float64 { return r.margin() / r.Buy.Buy * 100 }
	// Bars use ROI, not margin: one rare high-value commodity would otherwise flatten every other bar.
	best := 0.0
	for _, r := range routes {
		best = max(best, roi(r))
	}
	var rows [][]string
	for _, r := range routes {
		rows = append(rows, []string{
			bold.Render(r.Buy.Commodity),
			place.Render(truncate(r.Buy.Terminal, nameWidth)) + faint.Render(" "+r.Buy.System),
			cost.Render(money(r.Buy.Buy)),
			stock(r.Buy.Stock),
			place.Render(truncate(r.Sell.Terminal, nameWidth)) + faint.Render(" "+r.Sell.System),
			gain.Render(money(r.Sell.Sell)),
			signed(r.margin()),
			faint.Render(fmt.Sprintf("%.0f%%", roi(r))),
			bar(roi(r), best),
		})
	}
	columns := []column{
		{"commodity", false}, {"buy at", false}, {"aUEC", true}, {"scu", true},
		{"sell at", false}, {"aUEC", true}, {"margin", true}, {"roi", true}, {"", false},
	}
	return title + "\n" + layout(columns, rows)
}

// target renders a sell destination as where, price, and margin over buyPrice.
func target(p *price, buyPrice float64, withSystem bool) []string {
	if p == nil {
		return []string{faint.Render(none), "", ""}
	}
	where := place.Render(truncate(p.Terminal, nameWidth))
	if withSystem {
		where += faint.Render(" " + p.System)
	}
	return []string{where, gain.Render(money(p.Sell)), signed(p.Sell - buyPrice)}
}

func heading(title, detail string) string {
	text := accent.Render("▌ " + title)
	if detail != "" {
		text += faint.Render("  " + detail)
	}
	return text
}

func layout(columns []column, rows [][]string) string {
	widths := make([]int, len(columns))
	for index, c := range columns {
		widths[index] = len(c.header)
	}
	for _, row := range rows {
		for index, cell := range row {
			widths[index] = max(widths[index], lipgloss.Width(cell))
		}
	}
	line := func(cells []string) string {
		parts := make([]string, len(cells))
		for index, cell := range cells {
			align := lipgloss.Left
			if columns[index].right {
				align = lipgloss.Right
			}
			parts[index] = lipgloss.NewStyle().Width(widths[index]).Align(align).Render(cell)
		}
		return strings.TrimRight("  "+strings.Join(parts, "  "), " ")
	}
	headers := make([]string, len(columns))
	for index, c := range columns {
		headers[index] = faint.Render(c.header)
	}
	lines := []string{line(headers)}
	for _, row := range rows {
		lines = append(lines, line(row))
	}
	return strings.Join(lines, "\n")
}

// bar draws value relative to maximum using eighth blocks.
func bar(value, maximum float64) string {
	if value <= 0 || maximum <= 0 {
		return ""
	}
	eighths := max(1, int(value/maximum*barWidth*8))
	text := strings.Repeat("█", eighths/8)
	if remainder := eighths % 8; remainder > 0 {
		text += string([]rune("▏▎▍▌▋▊▉")[remainder-1])
	}
	return gain.Render(text)
}

func signed(value float64) string {
	if value < 0 {
		return loss.Render("-" + money(-value))
	}
	return gain.Render("+" + money(value))
}

func stock(scu float64) string {
	if scu <= 0 {
		return faint.Render("0")
	}
	return money(scu)
}

func money(value float64) string {
	digits := strconv.FormatInt(int64(value), 10)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}

func truncate(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	return string(runes[:width-1]) + "…"
}

func maxOf(values []float64) float64 {
	result := 0.0
	for _, value := range values {
		result = max(result, value)
	}
	return result
}

func nonEmpty(values ...string) []string {
	var result []string
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
