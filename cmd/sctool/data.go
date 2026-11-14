package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	apiBase    = "https://api.uexcorp.space/2.0/"
	staleAfter = time.Minute
)

type price struct {
	TerminalID int     `json:"id_terminal"`
	Commodity  string  `json:"commodity_name"`
	Terminal   string  `json:"terminal_name"`
	Buy        float64 `json:"price_buy"`
	Sell       float64 `json:"price_sell"`
	Stock      float64 `json:"scu_buy"`
	System     string  `json:"-"`
}

type terminal struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayname"`
	Nickname    string `json:"nickname"`
	System      string `json:"star_system_name"`
}

type snapshot struct {
	FetchedAt time.Time  `json:"fetched_at"`
	Prices    []price    `json:"prices"`
	Terminals []terminal `json:"terminals"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	return filepath.Join(base, "sctool"), err
}

// load returns cached data while it is fresh, otherwise refetches and archives the old snapshot.
// On fetch failure it returns the stale snapshot alongside the error so the UI can keep working.
func load() (snapshot, error) {
	dir, err := cacheDir()
	if err != nil {
		return snapshot{}, err
	}
	current := filepath.Join(dir, "current.json")
	cached, cacheErr := readSnapshot(current)
	if cacheErr == nil && time.Since(cached.FetchedAt) < staleAfter {
		return withSystems(cached), nil
	}
	fresh, err := fetchSnapshot()
	if err != nil {
		if cacheErr == nil {
			return withSystems(cached), fmt.Errorf("refresh failed, showing data from %s: %w", cached.FetchedAt.Format(time.Kitchen), err)
		}
		return snapshot{}, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "archive"), 0o755); err != nil {
		return snapshot{}, err
	}
	if cacheErr == nil {
		// Rename overwrites, so each day keeps only its last snapshot.
		archived := filepath.Join(dir, "archive", cached.FetchedAt.Format(time.DateOnly)+".json")
		if err := os.Rename(current, archived); err != nil {
			return snapshot{}, err
		}
	}
	return withSystems(fresh), writeSnapshot(current, fresh)
}

func readSnapshot(path string) (snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return snapshot{}, err
	}
	var s snapshot
	return s, json.Unmarshal(data, &s)
}

func writeSnapshot(path string, s snapshot) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func fetchSnapshot() (snapshot, error) {
	var s snapshot
	if err := fetch("commodities_prices_all", &s.Prices); err != nil {
		return snapshot{}, err
	}
	if err := fetch("terminals", &s.Terminals); err != nil {
		return snapshot{}, err
	}
	s.FetchedAt = time.Now()
	return s, nil
}

func fetch(endpoint string, into any) error {
	response, err := httpClient.Get(apiBase + endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", endpoint, response.StatusCode)
	}
	var body struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return fmt.Errorf("%s: %w", endpoint, err)
	}
	if body.Status != "ok" {
		return fmt.Errorf("%s: %s %s", endpoint, body.Status, body.Message)
	}
	return json.Unmarshal(body.Data, into)
}

func withSystems(s snapshot) snapshot {
	systems := make(map[int]string, len(s.Terminals))
	for _, t := range s.Terminals {
		systems[t.ID] = t.System
	}
	prices := make([]price, len(s.Prices))
	for index, p := range s.Prices {
		p.System = systems[p.TerminalID]
		if p.System == "" {
			p.System = "Unknown"
		}
		prices[index] = p
	}
	s.Prices = prices
	return s
}
