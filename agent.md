# cfctl — Cloudflare TUI Manager

## Project Goal

A terminal UI application to manage Cloudflare resources interactively, backed by the Cloudflare API.

---

## Tech Stack

**Go + Bubbletea (charm.sh)** — confirmed.

Single distributable binary, fast startup, native concurrency for API calls, and the charm.sh ecosystem (bubbletea + lipgloss + bubbles) is the right fit for this tool.

---

## What We Can Manage (Cloudflare API Scope)

> Note: API **Tokens** (not legacy API Keys) are recommended. Tokens support fine-grained per-resource permissions. Legacy keys sunset Sept 30, 2026.

### Zone Management
- List, create, delete zones
- Zone settings (SSL mode, security level, cache TTL, minification, etc.)
- Pause / unpause zones

### DNS
- List, create, update, delete DNS records (A, AAAA, CNAME, MX, TXT, SRV, etc.)
- Proxied vs DNS-only toggle

### Firewall & Security
- Firewall rules (list, create, edit, delete)
- WAF managed ruleset overrides
- IP Access Rules (allow/block/challenge)
- Rate limiting rules
- Bot Fight Mode toggle

### Workers
- List and view Workers scripts
- Deploy / delete Workers
- View Worker routes
- KV namespace management (list, read, write, delete keys)
- D1 database list and query

### Pages
- List Pages projects and deployments
- Trigger new deployments
- View build logs

### R2 Object Storage
- List buckets
- List objects in a bucket
- Upload / delete objects

### Load Balancers
- List load balancers, pools, origins
- Toggle pool health checks

### SSL / TLS
- View certificate packs
- Order / delete certificates
- Edge certificates settings

### Email Routing
- List routing rules
- Enable / disable email routing

### Tunnels (Cloudflare Tunnel / WARP)
- List tunnels and their connectors
- View tunnel routes

### Analytics
- Zone analytics (requests, bandwidth, threats, cached ratio)
- Workers analytics

### Account & Users
- Account details
- API token management (list, create, delete)
- Audit logs

---

## Planned Feature Scope (MVP vs Full)

### MVP (Phase 1)
- [ ] Auth setup: store API token securely (`~/.config/cfctl/config.toml`)
- [ ] Zone list & switch (main navigation context)
- [ ] DNS records: CRUD
- [ ] Firewall IP Access Rules: list / add / delete
- [ ] SSL/TLS: view cert status and mode

### Phase 2
- [ ] Workers: list, view routes, KV namespace CRUD
- [ ] R2: bucket + object browser
- [ ] Analytics dashboard (zone traffic overview)
- [ ] Page rules management

### Phase 3
- [ ] Pages deployments
- [ ] D1 database browser
- [ ] Tunnels management
- [ ] Email routing
- [ ] Audit log viewer

---

## Proposed UX Layout

```
┌─────────────────────────────────────────────────────────┐
│ cfctl  ▸ zone: example.com            [?] help  [q] quit│
├──────────────┬──────────────────────────────────────────┤
│ ZONES        │  DNS Records                             │
│ ▸ example.com│  ┌────────────────────────────────────┐  │
│   foo.dev    │  │ Type  Name        Content    Proxy  │  │
│   bar.io     │  │ A     @           1.2.3.4    ✓      │  │
│              │  │ CNAME www         example.com ✓     │  │
│ SECTIONS     │  │ MX    @           mail.ex.com 5     │  │
│ ▸ DNS        │  └────────────────────────────────────┘  │
│   Firewall   │  [n] new  [e] edit  [d] delete  [r] refresh│
│   SSL/TLS    │                                          │
│   Workers    │                                          │
│   Analytics  │                                          │
└──────────────┴──────────────────────────────────────────┘
```

---

## Project Structure (Go)

```
cfctl/
├── main.go
├── cmd/           # cobra CLI entry (flags, version)
├── api/           # Cloudflare API client wrappers
│   ├── client.go
│   ├── zones.go
│   ├── dns.go
│   ├── firewall.go
│   └── ...
├── tui/           # Bubbletea models
│   ├── app.go     # root model
│   ├── zones.go
│   ├── dns.go
│   └── ...
├── config/        # config file read/write
└── styles/        # lipgloss style definitions
```

---

## Dependencies (Go)

```
github.com/cloudflare/cloudflare-go   # official CF SDK
github.com/charmbracelet/bubbletea    # TUI framework
github.com/charmbracelet/bubbles      # TUI components (table, list, input, spinner)
github.com/charmbracelet/lipgloss     # styling
github.com/spf13/cobra                # CLI entry
github.com/spf13/viper                # config management
```

---

## Decisions Log

| Question | Decision |
|----------|----------|
| Language | Go + Bubbletea |
| Auth | API Token only (`cfut_` prefix / Cloudflare User Token format) |
| Config location | `~/.config/cfctl/config.toml` (XDG-style) |
| MVP scope | Phase 1: DNS + Firewall + SSL |
| Distribution | TBD — build from source for now |

---

## Build Order (Phase 1)

1. `go mod init` + project scaffold
2. `config/` — token storage, load/save `~/.config/cfctl/config.toml`
3. First-run wizard — prompt for token if config missing, validate against API
4. `api/` — thin wrappers: zones, DNS records, firewall IP rules, SSL cert info
5. `tui/` — root app model, zone selector sidebar, section switcher
6. DNS view — table, create/edit/delete forms
7. Firewall view — IP access rules table, add/delete
8. SSL/TLS view — cert pack list, mode display
9. Polish: keybindings help bar, error toasts, loading spinners
