package main

import (
	"slices"
	"testing"
)

var fixture = []price{
	{TerminalID: 1, Terminal: "Alpha", System: "Stanton", Commodity: "Gold", Buy: 100, Stock: 50},
	{TerminalID: 2, Terminal: "Beta", System: "Stanton", Commodity: "Gold", Buy: 90, Stock: 10},
	{TerminalID: 5, Terminal: "Empty", System: "Stanton", Commodity: "Gold", Buy: 80},
	{TerminalID: 3, Terminal: "Gamma", System: "Stanton", Commodity: "Gold", Sell: 120},
	{TerminalID: 4, Terminal: "Delta", System: "Pyro", Commodity: "Gold", Sell: 150},
	{TerminalID: 4, Terminal: "Delta", System: "Pyro", Commodity: "Quantanium", Sell: 300},
	{TerminalID: 1, Terminal: "Alpha", System: "Stanton", Commodity: "Iodine", Buy: 5, Stock: 7},
}

func TestTopMatches(t *testing.T) {
	names := []string{"Gold", "Gold Ore", "Quantanium", "Raw Quantanium", "Agricium", "Iodine"}
	identity := func(s string) []string { return []string{s} }
	cases := []struct {
		query string
		want  []string
	}{
		{"gold", []string{"Gold"}},
		{"quant", []string{"Quantanium", "Raw Quantanium"}},
		{"qtm", []string{"Quantanium", "Raw Quantanium"}},
		{"zzz", nil},
		{"", nil},
	}
	for _, c := range cases {
		if got := topMatches(c.query, names, identity, 3); !slices.Equal(got, c.want) && !(len(got) == 0 && c.want == nil) {
			t.Errorf("topMatches(%q) = %v, want %v", c.query, got, c.want)
		}
	}
}

func TestCommodityReport(t *testing.T) {
	lines := commodityReport(fixture, "Gold")
	if len(lines) != 2 || lines[0].System != "Pyro" || lines[1].System != "Stanton" {
		t.Fatalf("systems = %+v", lines)
	}
	if lines[0].Buy != nil || lines[0].Sell.Terminal != "Delta" {
		t.Errorf("Pyro = buy %v sell %v", lines[0].Buy, lines[0].Sell)
	}
	if lines[1].Buy.Terminal != "Beta" || lines[1].Sell.Terminal != "Gamma" {
		t.Errorf("Stanton = buy %v sell %v", lines[1].Buy, lines[1].Sell)
	}
}

func TestLocationReport(t *testing.T) {
	lines := locationReport(fixture, 1)
	if len(lines) != 2 || lines[0].Item.Commodity != "Gold" || lines[1].Item.Commodity != "Iodine" {
		t.Fatalf("lines = %+v", lines)
	}
	if lines[0].InSystem.Terminal != "Gamma" || lines[0].Anywhere.Terminal != "Delta" {
		t.Errorf("gold sells = %v / %v", lines[0].InSystem, lines[0].Anywhere)
	}
	if lines[1].InSystem != nil || lines[1].Anywhere != nil {
		t.Errorf("iodine should have no sell targets")
	}
}
