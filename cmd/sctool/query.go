package main

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"
)

const exact = 100000

// fuzzyScore ranks exact > substring > ordered subsequence; -1 means no match.
func fuzzyScore(query, candidate string) int {
	q, c := strings.ToLower(strings.TrimSpace(query)), strings.ToLower(candidate)
	switch {
	case q == "" || c == "":
		return -1
	case q == c:
		return exact
	case strings.HasPrefix(c, q):
		return 50000 - len(c)
	case strings.Contains(c, q):
		return 20000 - len(c)
	}
	score, from, previous := 0, 0, -2
	for _, r := range q {
		offset := strings.IndexRune(c[from:], r)
		if offset < 0 {
			return -1
		}
		position := from + offset
		score++
		if position == previous+1 {
			score += 5
		}
		if position == 0 || c[position-1] == ' ' || c[position-1] == '-' {
			score += 3
		}
		previous, from = position, position+utf8.RuneLen(r)
	}
	return score*10 - len(c)
}

// topMatches returns up to n items ranked by their best-matching key, keeping only exact matches if any exist.
// Ties keep input order.
func topMatches[T any](query string, items []T, keys func(T) []string, n int) []T {
	type scored struct {
		item  T
		score int
	}
	var ranked []scored
	for _, item := range items {
		best := -1
		for _, key := range keys(item) {
			best = max(best, fuzzyScore(query, key))
		}
		if best >= 0 {
			ranked = append(ranked, scored{item, best})
		}
	}
	slices.SortStableFunc(ranked, func(a, b scored) int { return cmp.Compare(b.score, a.score) })
	if len(ranked) > 0 && ranked[0].score == exact {
		ranked = slices.DeleteFunc(ranked, func(r scored) bool { return r.score != exact })
	}
	result := make([]T, 0, n)
	for _, r := range ranked[:min(n, len(ranked))] {
		result = append(result, r.item)
	}
	return result
}

func commodityNames(prices []price) []string {
	names := make([]string, 0, len(prices))
	for _, p := range prices {
		names = append(names, p.Commodity)
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// tradingTerminals returns terminals with price data, busiest first, so "new babbage" ranks TDD above the ammo shop.
func tradingTerminals(s snapshot) []terminal {
	rows := map[int]int{}
	for _, p := range s.Prices {
		rows[p.TerminalID]++
	}
	result := slices.DeleteFunc(slices.Clone(s.Terminals), func(t terminal) bool { return rows[t.ID] == 0 })
	slices.SortFunc(result, func(a, b terminal) int {
		return cmp.Or(cmp.Compare(rows[b.ID], rows[a.ID]), cmp.Compare(a.Name, b.Name))
	})
	return result
}

func best(prices []price, keep func(price) bool, better func(a, b price) bool) *price {
	var found *price
	for index := range prices {
		if keep(prices[index]) && (found == nil || better(prices[index], *found)) {
			found = &prices[index]
		}
	}
	return found
}

// cheaper prefers terminals that actually have stock, then the lower price.
func cheaper(a, b price) bool {
	if (a.Stock > 0) != (b.Stock > 0) {
		return a.Stock > 0
	}
	return a.Buy < b.Buy
}
func pricier(a, b price) bool { return a.Sell > b.Sell }

type systemLine struct {
	System    string
	Buy, Sell *price
}

func commodityReport(prices []price, commodity string) []systemLine {
	var systems []string
	for _, p := range prices {
		if p.Commodity == commodity {
			systems = append(systems, p.System)
		}
	}
	slices.Sort(systems)
	var lines []systemLine
	for _, system := range slices.Compact(systems) {
		here := func(p price) bool { return p.Commodity == commodity && p.System == system }
		lines = append(lines, systemLine{
			System: system,
			Buy:    best(prices, func(p price) bool { return here(p) && p.Buy > 0 }, cheaper),
			Sell:   best(prices, func(p price) bool { return here(p) && p.Sell > 0 }, pricier),
		})
	}
	return lines
}

type locationLine struct {
	Item               price
	InSystem, Anywhere *price
}

func locationReport(prices []price, terminalID int) []locationLine {
	var lines []locationLine
	for _, item := range prices {
		if item.TerminalID != terminalID || item.Buy <= 0 {
			continue
		}
		sells := func(p price) bool { return p.Commodity == item.Commodity && p.Sell > 0 }
		lines = append(lines, locationLine{
			Item:     item,
			InSystem: best(prices, func(p price) bool { return sells(p) && p.System == item.System }, pricier),
			Anywhere: best(prices, sells, pricier),
		})
	}
	slices.SortFunc(lines, func(a, b locationLine) int { return cmp.Compare(a.Item.Commodity, b.Item.Commodity) })
	return lines
}

type route struct{ Buy, Sell price }

func (r route) margin() float64 { return r.Sell.Sell - r.Buy.Buy }

// bestRoute returns a commodity's cheapest in-stock buy and best sell anywhere; either may be nil.
func bestRoute(prices []price, name string) (buy, sell *price) {
	buy = best(prices, func(p price) bool { return p.Commodity == name && p.Buy > 0 && p.Stock > 0 }, cheaper)
	sell = best(prices, func(p price) bool { return p.Commodity == name && p.Sell > 0 }, pricier)
	return buy, sell
}

// topRoutes pairs each commodity's cheapest in-stock buy with its best sell anywhere, highest margin first.
func topRoutes(prices []price, n int) []route {
	var routes []route
	for _, name := range commodityNames(prices) {
		buy, sell := bestRoute(prices, name)
		if buy != nil && sell != nil && sell.Sell > buy.Buy {
			routes = append(routes, route{*buy, *sell})
		}
	}
	slices.SortFunc(routes, func(a, b route) int { return cmp.Compare(b.margin(), a.margin()) })
	return routes[:min(n, len(routes))]
}
