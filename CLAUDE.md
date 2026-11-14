# CLAUDE.md

This repo is a collection of Star Citizen CLI tools. Each tool is a `main` package under `cmd/<name>/`, and the binary takes the directory's name. The repo is consumed by a NixOS config through `buildGoModule`, which builds everything under `cmd/`. A new tool needs no Nix changes, but keep `vendorHash` in mind whenever dependencies change.

When a second tool needs the UEX fetch and cache code, move it from `cmd/sctool/data.go` into `internal/uex/` rather than copying it.

## sctool (`cmd/sctool/`)

An interactive TUI for looking up commodity prices.

- `data.go`: fetches from the UEX API, caches to `~/.cache/sctool/current.json`, and archives one snapshot per day to `archive/YYYY-MM-DD.json`. Data is refetched at most once every `staleAfter` (1 minute). `withSystems` joins prices to terminals to fill in `price.System`, because the prices endpoint has no system field.
- `query.go`: pure functions with no I/O: `fuzzyScore`, the generic `topMatches`, `commodityReport`, `locationReport` and `topRoutes` (the landing view). Put new logic here and test it in `query_test.go`.
- `view.go`: renders the views as aligned rows (no tables). It only uses ANSI palette colors 0-15 and bold/faint/reverse so it follows the terminal theme. Never use hex colors.
- `main.go`: the TUI, built with bubbletea, bubbles (`textinput`, `viewport`), lipgloss and `lipgloss/table`. It has vim-style NORMAL and INSERT modes. Searches run as an async `tea.Cmd` that calls `load()`.

## UEX field meanings

- `price_buy` is what the player pays the terminal. `scu_buy` is the stock available to buy.
- `price_sell` is what the terminal pays the player.
- `0` means the terminal doesn't trade that commodity.
- API docs: https://uexcorp.space/api/documentation

## Decisions

- No SQLite: about 3.4k rows fits easily in memory and lookups take microseconds.
- No fuzzy-matching dependency: `fuzzyScore` is about 25 lines.
- Exact matches hide fuzzy matches.
- Terminals are ranked busiest first, so `new babbage` finds TDD before the shops.
- When picking the cheapest buy, terminals with stock win over cheaper ones showing 0 SCU.

## Commands

```sh
go test ./...
go vet ./...
go build ./cmd/...
```

## Sandbox note

In the Claude Code sandbox, `~/.cache/go-build`, `~/go/pkg` and `~/go/bin` are read-only:
- Prefix commands with `GOCACHE=$TMPDIR/gocache`.
- For `go get`, also set `GOPATH=$TMPDIR/gopath GOMODCACHE=$HOME/go/pkg/mod`.
- `go mod tidy` fails because it wants test-only dependencies that aren't in the read-only module cache. Use `go get <package>@<version>` instead.
