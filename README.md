# alfred-1password-search

Search 1Password items from Alfred and copy passwords, usernames and one-time codes.

## Requirements

- [Alfred 6](https://www.alfredapp.com/) with Powerpack (the details window is an Alfred UI view)
- [1Password CLI](https://developer.1password.com/docs/cli/) (`op`) with the 1Password app integration turned on

## Installation

1. Download the latest `.alfredworkflow` from [Releases](https://github.com/rwilgaard/alfred-1password-search/releases)
2. Double-click to import into Alfred
3. Trigger `op`. The first run fetches your items from 1Password, so unlock with Touch ID if asked

## Usage

### `op [query]`: search all vaults

Matches on title, URL, vault and category. Items you use often rank higher, and the last used item shows on top for five minutes.

| Action | Result (default, configurable) |
|--------|--------|
| ⏎ | Copy Password |
| ⌘⏎ | Show Details |
| ⌥⏎ | Copy Username |
| ⌃⏎ | Open URL |
| ⇧⏎ | Copy OTP |

Each key can be set to Copy Password, Copy Username, Copy OTP, Open URL or Show Details.
Copied values are hidden from clipboard history and cleared after `clipboard_timeout` seconds.

The item list is cached and updates in the background when 1Password changes. If 1Password is locked, an "Update items" row lets you unlock and update.

### `ops [query]`: search selected vaults

Same as `op`, limited to the vaults in `selected_vaults`. The keyword is set with `selected_vaults_keyword`.

### `opv [query]`: pick a vault

Choose a vault, then search only that vault.

### `opg`: generate a password

Shows one password with symbols and one without, `pw_length` characters long. ⏎ copies it.

### Hotkeys

| Hotkey | Result |
|--------|--------|
| ⌘⇧L | Search all vaults |
| ⌘⇧P | Search selected vaults |
| ⌃⌥P | Show details of the last used item |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `selected_vaults` | — | Comma-separated vault names for the selected vaults keyword |
| `selected_vaults_keyword` | `ops` | Keyword for searching selected vaults |
| `modifier_return` | `Copy Password` | Action for ⏎ |
| `modifier_cmd` | `Show Details` | Action for ⌘⏎ |
| `modifier_opt` | `Copy Username` | Action for ⌥⏎ |
| `modifier_ctrl` | `Open URL` | Action for ⌃⏎ |
| `modifier_shift` | `Copy OTP` | Action for ⇧⏎ |
| `fuzzy_search` | on | Use fuzzy matching when filtering items |
| `clipboard_timeout` | `30` | Seconds before a copied value is cleared (if unchanged); `0` disables |
| `check_for_updates` | on | Check GitHub for a new version once a day |
| `allowed_symbols` | `!@#$%^&*` | Symbols used in generated passwords |
| `pw_length` | `32` | Default length for generated passwords |

## Development

```sh
make build          # build arch-specific binaries
make dev            # build, install into Alfred in place, and reload
make package-alfred # build + zip into .alfredworkflow
make zip-alfred     # zip into .alfredworkflow without rebuilding (for CI/CD)
make fmt            # format with gofumpt
make release V=x.y.z # bump version, package, and tag locally
```

Requires Go 1.27.4+ and `golangci-lint`.
