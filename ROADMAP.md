# Roadmap

## Phase 1 — CLI Foundation ✅

### 1.1 Project setup ✅
- [x] Cobra dependency
- [x] `cmd/frf/main.go` entry point
- [x] `internal/cli/root.go` with global flags (`--token`, `--base-url`)
- [x] Token-based auth (`NewWithToken`, `FREEFEED_APP_TOKEN`)

### 1.2 Client expansion ✅
- [x] `GetTimeline` / `GetPost` (existed)
- [x] `SearchPosts`
- [x] `WhoAmI` / `GetUserProfile`
- [x] `GetSubscribers` / `GetSubscriptions`
- [x] `GetMyGroups`
- [x] `CreatePost` / `CreateDirectPost`
- [x] `UpdatePost` / `DeletePost`
- [x] `LikePost` / `UnlikePost` / `HidePost` / `UnhidePost`
- [x] `AddComment` / `UpdateComment` / `DeleteComment`
- [x] `Subscribe` / `Unsubscribe`

### 1.3 CLI commands ✅
- [x] `frf timeline` (home/discussions/directs/posts/likes/comments, --limit/--offset/--page)
- [x] `frf post get/create/update/delete/like/unlike/hide/unhide`
- [x] `frf comment add/update/delete`
- [x] `frf direct create --to`
- [x] `frf search`
- [x] `frf user me/profile/subscribers/subscriptions/subscribe/unsubscribe`
- [x] `frf group list/timeline`

### 1.4 Output ✅
- [x] Text-only output (no JSON mode — text works for humans and agents)
- [x] Clean post/comment/profile/user list formatters

---

## Phase 2 — Testing & Quality

- [ ] Unit tests for client methods (httptest fixtures)
- [ ] Test fixtures in `internal/client/testdata/`
- [ ] Edge case tests (array vs map API responses, empty responses)
- [ ] `go vet` / `staticcheck` clean

---

## Phase 3 — Claude Code Integration

- [ ] Update skill files with real paths and tested examples
- [ ] `freefeed-digest` agent — summarize unread feed
- [ ] `freefeed-reply` agent — draft replies
- [ ] Dev agents: `expand-client`, `add-cli-command`, `test-writer` (templates exist)

---

## Phase 4 — TUI Improvements

- [ ] Post creation (compose view)
- [ ] Comment creation (inline)
- [ ] Like/unlike from detail view
- [ ] Search view
- [ ] User profile view
- [ ] Group browser (replace hardcoded overlay)
- [ ] Direct messages compose
- [ ] Real notification data
- [ ] Configurable keybindings / themes

---

## Phase 5 — Advanced

- [ ] Realtime updates (FreeFeed websocket)
- [ ] Attachment upload (`frf post create --attach file.jpg`)
- [ ] Image preview in terminal (sixel/kitty)
- [ ] Multi-account support
- [ ] Opt-out filtering (`#noai`, `#opt-out-ai` tags — port from MCP server)
- [ ] Offline cache
