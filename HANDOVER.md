# Handover

## What Is This

Go-based FreeFeed client with two interfaces:
- **`frf`** — CLI tool (primary, text output, for humans and agents)
- **`frf-tui`** — TUI client (secondary, Bubble Tea)

Part of the `coriolis` project family. Sibling: `../freefeed-mcp-server/` (Python MCP server).

## Current State (2026-04-14)

**Phase 1 complete.** Full CLI with read + write operations, tested on live API.

### What works

**CLI (`cmd/frf/`)** — 852 lines across 9 files in `internal/cli/`:
- All timeline types: home, discussions, directs, user posts/likes/comments
- Post CRUD: get, create, update, delete
- Post actions: like, unlike, hide, unhide
- Comments: add, update, delete
- Direct messages: create with recipients
- Search with operators (from:, intitle:, incomment:)
- User: whoami, profile, subscribers, subscriptions, subscribe, unsubscribe
- Groups: list (120 groups), timeline
- Pagination: `--limit`, `--offset`, `--page`
- Auth: token (`--token` / `FREEFEED_APP_TOKEN`) or username+password (`.env`)

**API Client** (`internal/client/client.go`) — 1139 lines:
- 20+ methods covering full FreeFeed API v4
- Handles FreeFeed's inconsistent response formats (array vs map, single object vs collection)
- Token and password auth

**TUI (`internal/app/model.go`)** — 1185 lines:
- Two-panel layout, section navigation, comment pagination, auto-refresh, infinite scroll
- Read-only — doesn't use any write client methods yet

### Design decisions
- **Text-only output** — no JSON mode. Plain text is readable by humans and parseable by agents. Removed JSON/format flag during code review.
- **No intermediate types** — CLI prints directly from `client.*` types via `output.go` helpers
- **Two binaries, shared core** — `cmd/frf/` and `cmd/frf-tui/` share `internal/client/` and `internal/app/config.go`

### What's NOT done
- **No tests** — no unit tests, no fixtures
- **No attachments** — upload/download not implemented
- **No `.gitignore`** — `bin/`, `.env` not ignored
- **TUI is read-only** — doesn't create posts, comments, or do any write operations
- **TUI has hardcoded groups** overlay (model.go:502-518)
- **TUI sections without data**: Saved, Best, Notifications
- **Skills/agents** exist as templates but haven't been tested with real CLI

### Known quirks
- FreeFeed API returns `posts` as single object on create, array on timeline, map on post get — client handles all three
- Timestamps are Unix milliseconds as strings
- `subscribers` in whoami response contains ALL related users (subscribers + subscriptions), groups filtered by `type == "group"`
- Can't like own posts (API returns 403) — correct behavior

## Key Files

| File | Lines | What |
|------|-------|------|
| `internal/client/client.go` | 1139 | API client — complete |
| `internal/app/model.go` | 1185 | TUI model — read-only |
| `internal/cli/output.go` | 130 | Text output helpers |
| `internal/cli/post.go` | 162 | Post commands (largest CLI file) |
| `internal/app/config.go` | 55 | Shared config |

## Next Steps (priority order)

1. **Tests** — unit tests for client methods with httptest fixtures
2. **Claude Code integration** — update skills with real paths, test agents
3. **TUI write operations** — post create, comment, like/unlike
4. **Attachments** — upload in CLI, view in TUI
5. **Opt-out filtering** — port from MCP server
