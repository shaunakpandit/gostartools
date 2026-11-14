# gostartools

Star Citizen command-line tools written in Go.

| Tool | Description |
|---|---|
| `sctool` | Interactive lookup of commodity prices by commodity or by location, using live data from the [UEX Corp API](https://uexcorp.space) |

## Install

```sh
go install ./cmd/...         # installs every tool (sctool, ...) into ~/go/bin
go build ./cmd/sctool        # or build one tool in the current directory
```

Each directory under `cmd/` is one binary, named after its directory.

### Nix

`buildGoModule` builds every `cmd/*` binary by default. Use `subPackages = [ "cmd/sctool" ];` to pick individual tools. Set `vendorHash` to a real hash: build once with `lib.fakeHash` and copy the `got:` value from the error.

## sctool

Run `sctool`. It starts in NORMAL mode.

| Key | Action |
|---|---|
| `c` | Search a commodity |
| `l` | Search a location |
| `/` | Search again in the current mode |
| `enter` / `esc` | Run the search / cancel |
| `j` / `k` | Scroll |
| `ctrl-d` / `ctrl-u` | Half page down / up |
| `gg` / `G` | Top / bottom |
| `?` | Help |
| `q` | Quit |

Search is fuzzy (for example, `quant` finds Quantainium) and shows the top 3 matches. If something matches exactly, only the exact matches are shown.

- **Commodity search:** a table per commodity with one row per star system. It shows the cheapest place to buy (price and stock in SCU) and the best place to sell.
- **Location search:** the commodities you can buy at that terminal, each shown as `name - price/SCU - stock SCU`. Next to each one is the best place to sell it in the same system and the best place to sell it anywhere.

All prices are aUEC per SCU.

### Data

- The tool fetches `commodities_prices_all` and `terminals` from `api.uexcorp.space/2.0` at most once a minute. Every search checks the age of the data first.
- The current snapshot is cached at `~/.cache/sctool/current.json`.
- When the snapshot is replaced, the old one is moved to `~/.cache/sctool/archive/YYYY-MM-DD.json`. Each day keeps only its last snapshot, which you can use for price trends later.
- If a refresh fails, the tool keeps showing the cached data and puts the error in the status bar.

## Development

```sh
go test ./...
```

To add a tool, create `cmd/<name>/main.go`.
