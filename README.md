# cfctl

A terminal UI for managing Cloudflare — DNS records, firewall rules, and SSL certificates, all from your terminal.

```
╭──────────────────╮╭──────────────────────────────────────────────────────────────────╮
│ ZONES            ││ DNS Records                                                      │
│ ▸ gite.io        ││                                                                  │
│   example.com    ││   TYPE    NAME                  CONTENT            TTL    PROXY  │
│                  ││   ────────────────────────────────────────────────────────────── │
│ SECTIONS         ││ ▸ A       gite.io               188.245.60.83      Auto   ✗      │
│ ▸ [1] DNS        ││   CNAME   www                   gite.io            Auto   ✗      │
│   [2] Firewall   ││   MX      gite.io               mail.gite.io       3600   ✗      │
│   [3] SSL/TLS    ││   TXT     _dmarc                v=DMARC1; p=none   Auto   ✗      │
│                  ││                                                                  │
│ j/k navigate     ││ [n] new  [e] edit  [d] delete  [r] refresh  [j/k] navigate       │
│ l/enter select   ││                                                                  │
╰──────────────────╯╰──────────────────────────────────────────────────────────────────╯
```

## Features

### Phase 1 (current)
- **Multi-account support** — manage multiple Cloudflare API tokens; switch accounts on the fly with `[t]`
- **Zone switching** — navigate all your Cloudflare zones from the sidebar
- **DNS records** — list, create, edit, and delete records (A, AAAA, CNAME, MX, TXT, and more); toggle proxy mode per record
- **Firewall / IP Access Rules** — list, add, and delete allow/block/challenge rules
- **SSL/TLS** — view certificate packs and their status per zone
- **Activity log** — per-type filtered log of all actions taken in the session
- Vim-style navigation (`j`/`k`, `h`/`l`, `g`/`G`) throughout
- First-run token wizard with live validation

### Planned
- Workers, KV, D1 browser
- R2 object storage browser
- Zone analytics dashboard
- Pages deployments
- Tunnels management
- Email routing
- Audit log viewer

## Installation

### From source

```bash
git clone https://github.com/mrbooshehri/cfctl.git
cd cfctl
go build -o cfctl .
```

Move the binary somewhere on your `$PATH`:

```bash
mv cfctl ~/.local/bin/
```

### Requirements

- Go 1.21+
- A Cloudflare API token (see [Auth](#auth))

## Auth

cfctl uses **Cloudflare API Tokens** (not the legacy global API key).

1. Go to [dash.cloudflare.com](https://dash.cloudflare.com) → My Profile → API Tokens
2. Click **Create Token**
3. Use the *Edit zone DNS* template or build a custom token with the permissions you need
4. Copy the token — it starts with `cfut_`

On first run cfctl prompts for an account name and token, validates the token, and saves to:

```
~/.config/cfctl/config.toml
```

The file is created with `chmod 0600` (readable only by your user). To skip the file entirely, set the environment variable instead:

```bash
export CFCTL_TOKEN=cfut_...
```

### Multiple accounts

Press `t` from the main view to open the account manager. From there you can:

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate the account list |
| `enter` | Switch to the selected account |
| `a` | Add a new account (prompts for name + token) |
| `e` | Edit selected account (rename and/or replace token) |
| `d` | Delete selected account (with confirmation) |
| `esc` / `q` | Close account manager |

Config file format with multiple accounts:

```toml
active = "work"

[tokens]
  "personal" = "cfut_aaa..."
  "work"     = "cfut_bbb..."
```

Existing single-token configs are migrated automatically on first run.

## Usage

```bash
cfctl          # launch the TUI
cfctl version  # print version
```

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `h` | Focus sidebar |
| `l` / `enter` | Focus content panel |
| `tab` | Toggle between panels |
| `1` | Go to DNS section |
| `2` | Go to Firewall section |
| `3` | Go to SSL/TLS section |
| `4` | Go to Logs section |
| `t` | Open account manager |
| `q` / `ctrl+c` | Quit |

### DNS records

| Key | Action |
|-----|--------|
| `n` | New record |
| `e` | Edit selected record |
| `d` | Delete selected record |
| `r` | Refresh list |

In the new/edit form, `tab` / `↑↓` moves between fields, `space` toggles proxy mode, and `enter` advances to the next field (or submits on the last).

### Firewall — IP Access Rules

| Key | Action |
|-----|--------|
| `n` | New rule |
| `d` | Delete selected rule |
| `r` | Refresh list |

Supported modes: `block`, `challenge`, `whitelist`, `js_challenge`  
Supported targets: `ip`, `ip_range` (CIDR), `country` (2-letter code)

### SSL / TLS

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate certificate packs |
| `r` | Refresh |

## Project structure

```
cfctl/
├── main.go
├── cmd/              # cobra CLI entry (flags, version subcommand)
├── config/           # multi-account config load/save (~/.config/cfctl/config.toml)
├── api/              # thin wrappers around cloudflare-go
│   ├── client.go     # token auth + validation
│   ├── zones.go
│   ├── dns.go
│   ├── firewall.go
│   └── ssl.go
├── tui/              # bubbletea models
│   ├── app.go        # root model, layout, panel routing
│   ├── setup.go      # first-run account wizard
│   ├── accountmgr.go # multi-account manager (add/edit/delete/switch)
│   ├── dns.go        # DNS records view + form
│   ├── firewall.go
│   ├── ssl.go
│   └── logs.go       # activity log view
└── styles/           # lipgloss colour palette and shared styles
```

## Dependencies

| Package | Purpose |
|---------|---------|
| [cloudflare/cloudflare-go](https://github.com/cloudflare/cloudflare-go) | Official Cloudflare Go SDK |
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) | TUI components (textinput) |
| [spf13/cobra](https://github.com/spf13/cobra) | CLI entry point |
| [spf13/viper](https://github.com/spf13/viper) | Config file + env var handling |

## Contributing

Issues and pull requests are welcome.

For a new feature, open an issue first to discuss scope — check `agent.md` for the planned roadmap and architectural decisions made so far.

## License

MIT
