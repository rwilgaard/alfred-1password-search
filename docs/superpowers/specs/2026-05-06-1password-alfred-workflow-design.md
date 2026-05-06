# Alfred 1Password Search — Design Spec

**Date:** 2026-05-06  
**Repo:** `rwilgaard/alfred-1password-search`  
**Reference:** `rwilgaard/alfred-lastpass-search` (direct port)

---

## Overview

An Alfred workflow for searching and interacting with a 1Password vault. Full feature parity with the existing `alfred-lastpass-search` workflow, adapted for the `op` (1Password CLI) and the 1Password Mac app with CLI integration enabled. Authentication is handled entirely by the Mac app (Touch ID / app unlock state) — no explicit sign-in flow in the workflow.

---

## Architecture

```
src/
  main.go
  cmd/
    root.go         ← workflow init, app-unlock check, config
    list.go         ← `list` command (op keyword)
    show.go         ← `show` command (item detail view)
    vaults.go       ← `vaults` command (opv keyword)
    generate.go     ← `generate` command (opgen keyword)
    update.go       ← `update` command (auto-update check)
  pkg/
    onepassword/
      onepassword.go  ← service wrapping `op` CLI calls
    util/
      util.go         ← icons, regex helpers, password gen
workflow/
  info.plist        ← Alfred workflow config
  icons/            ← same icon set as alfred-lastpass-search
```

Same overall shape as `alfred-lastpass-search`. The `util` package is copied directly; `generate.go` and `update.go` are nearly identical.

---

## Service Layer

**File:** `src/pkg/onepassword/onepassword.go`

### Types

```go
type OnePasswordService struct {
    BinPath string
}

type OPVault struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type OPItem struct {
    ID    string  `json:"id"`
    Title string  `json:"title"`
    Vault OPVault `json:"vault"`
    URLs  []struct {
        Href string `json:"href"`
    } `json:"urls"`
    Fields []OPField `json:"fields"`
}

type OPField struct {
    ID      string `json:"id"`
    Label   string `json:"label"`
    Type    string `json:"type"`
    Value   string `json:"value"`
    Purpose string `json:"purpose"` // "USERNAME", "PASSWORD", or ""
}
```

### Methods

| Method | CLI call | Notes |
|--------|----------|-------|
| `NewOnePasswordService(binPath string)` | — | Validates binPath |
| `IsAvailable() bool` | `op account list --cache` | Returns false if exit code != 0 (app locked/not running) |
| `GetVaults() ([]OPVault, error)` | `op vault list --cache --format=json` | |
| `GetItems(query, vaultID string, fuzzy bool) ([]OPItem, error)` | `op item list --cache --format=json [--vault=<id>]` | Filters by query client-side; fuzzy handled by awgo |
| `GetItemDetails(id string) (*OPItem, error)` | `op item get <id> --cache --format=json` | Returns full field list |
| `CheckValidity(item OPItem, action string) bool` | — | Returns false if action requires a field that is empty |

All read commands pass `--cache` to use the `op` CLI's built-in cache, avoiding repeated round-trips to the 1Password app.

---

## Commands

### `root.go`

- Initialises `aw.Workflow` with `update.GitHub(repo)`, `aw.MaxResults(25)`, `aw.SuppressUIDs(true)`
- Calls `alfredutils.InitWorkflow` and `alfredutils.CheckForUpdates`
- Creates `OnePasswordService` with binary path `op`
- Calls `IsAvailable()`; if false, shows a single item: **"1Password is locked"** with subtitle **"Unlock the 1Password app first"**, `Valid(false)`, then calls `alfredutils.HandleFeedback` and returns
- Runs `rootCmd.Execute()`

Config struct:

```go
type workflowConfig struct {
    ModifierReturn      string `env:"modifier_return"`
    ModifierCmd         string `env:"modifier_cmd"`
    ModifierOpt         string `env:"modifier_opt"`
    ModifierCtrl        string `env:"modifier_ctrl"`
    AllowedSymbols      string `env:"allowed_symbols"`
    FuzzySearch         bool   `env:"fuzzy_search"`
    IntelligentOrdering bool   `env:"intelligent_ordering"`
}
```

Default modifier values (set in Alfred workflow User Configuration):
- `modifier_return` = `Copy Password`
- `modifier_cmd` = `Show Details`
- `modifier_opt` = `Copy Username`
- `modifier_ctrl` = `Copy ID`

### `list.go`

- Cobra command `list`, args: 0–1 (optional query), flag: `--vault` (string)
- Calls `GetItems(query, vaultFlag, cfg.FuzzySearch)`
- For each item, creates an Alfred result:
  - Title: `item.Title`
  - Subtitle: `<VaultName>  •  <Username>` (username extracted from field with `Purpose == "USERNAME"`)
  - Match: `<id> <vault> <title> <url> <username>`
  - Vars: `item_id`, `item_title`, `item_url`, `item_vault_id`, `query`, `action` (= `cfg.ModifierReturn`)
  - Modifiers: Ctrl, Opt, Cmd — same pattern as LastPass `list.go`
- If `cfg.FuzzySearch && len(query) > 0`, calls `wf.Filter(query)`
- Calls `alfredutils.HandleFeedback`

### `vaults.go`

- Cobra command `vaults`, args: exactly 1 (query for filtering vault list)
- Calls `GetVaults()`
- If `cfg.IntelligentOrdering`, disables `SuppressUIDs`
- Shows a "Select vault" item (not valid, matches `*`) followed by all vaults
- Calls `wf.Filter(args[0])`
- Var `vault_id` set on each vault item (used by Alfred to pass to `list --vault`)
- Calls `alfredutils.HandleFeedback`

### `show.go`

- Cobra command `show`, args: exactly 1 (item ID)
- Calls `GetItemDetails(itemID)`
- Excluded fields (skip entirely): fields with empty `Value`, and internal metadata fields (e.g. `Type == "ADDRESS"` sub-fields that are empty)
- Redacted field types: `CONCEALED` and `OTP` → show `••••••••` (32 bullets), `sensitive = "true"`
- All other non-empty fields are shown with their `Label` as title and `Value` as subtitle
- Fixed items at top: **"Go back"** (arg: `go_back`)
- Fixed items at bottom: **"Edit entry"** (arg: `edit`), **"Delete entry"** (arg: `delete`)
- Notes field: subtitle = "Press ↩ to show notes", arg = `notes`
- All other fields: subtitle = value (or redacted), arg = value, vars: `sensitive`, `field`
- Calls `alfredutils.HandleFeedback`

### `generate.go`

Identical to `alfred-lastpass-search/src/cmd/generate.go`, with subtitle changed from "add to LastPass" to "add to 1Password".

### `update.go`

Identical to `alfred-lastpass-search/src/cmd/update.go`.

---

## Keywords (Alfred workflow)

| Keyword | Command arg | Description |
|---------|-------------|-------------|
| `op [query]` | `list [query]` | Search entire vault |
| `opv [query]` | `vaults [query]` → `list --vault=<id> [query]` | Filter by vault, then search |
| `opgen [length]` | `generate --length=<n>` | Generate random password |
| `opadd` | — | Open 1Password app to add new item |

No `opsync` or `opout` — not applicable with Mac app integration.

---

## Module & Build Config

- **Module path:** `github.com/rwilgaard/alfred-1password-search`
- **Binary name:** `alfred-1password-search`
- **Dependencies:** same as LastPass (`awgo`, `cobra`, `go-alfredutils`, `go-password`, `fuzzy`)
- **Makefile:** replace `PLACEHOLDER` with `alfred-1password-search`
- **`.goreleaser.yaml`:** replace `PLACEHOLDER.alfredworkflow` with `alfred-1password-search.alfredworkflow`
- **`repo` const:** `rwilgaard/alfred-1password-search`

---

## Error Handling

- `IsAvailable()` failure → friendly locked message, workflow exits gracefully (no fatal error)
- `op` CLI not found → `wf.FatalError` with message pointing user to install 1Password CLI
- Any `op` command failure → `wf.FatalError(err)`

---

## Icons

Reuse the full icon set from `alfred-lastpass-search/workflow/icons/`. Field-to-icon mapping via `util.GetIcon(label)` (same function). The `icon.png` workflow icon should be updated to a 1Password-appropriate icon.
