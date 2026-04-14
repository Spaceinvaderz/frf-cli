# frf — FreeFeed CLI & TUI

## Vision

**Primary**: CLI tool (`frf`) — text output, readable by humans and agents alike.
**Secondary**: Interactive TUI client (`frf-tui`) for humans (Bubble Tea).

Both binaries share the API client and configuration.

## Reference Implementation

The Python MCP server at `../freefeed-mcp-server/` is the canonical reference for FreeFeed API integration:
- API endpoints and response shapes — `freefeed_mcp_server/client.py`
- Opt-out filtering logic — `freefeed_mcp_server/server.py`

## Architecture

```
cmd/
  frf/main.go               # CLI entry point (cobra)
  frf-tui/main.go            # TUI entry point (bubbletea)
internal/
  client/client.go           # FreeFeed API client (shared)
  app/
    config.go                # Env-based configuration (shared)
    model.go                 # Bubble Tea model (TUI only)
  cli/
    root.go                  # Cobra root command, newClient()
    output.go                # Text output helpers
    timeline.go              # frf timeline
    post.go                  # frf post get/create/update/delete/like/unlike/hide/unhide
    comment.go               # frf comment add/update/delete
    direct.go                # frf direct create
    search.go                # frf search
    user.go                  # frf user me/profile/subscribers/subscriptions/subscribe/unsubscribe
    group.go                 # frf group list/timeline
```

## CLI Commands

```bash
# Timeline
frf timeline                          # home feed
frf timeline discussions              # discussions
frf timeline directs                  # direct messages
frf timeline posts <username>         # user's posts
frf timeline likes <username>         # user's likes
frf timeline comments <username>      # user's comments
frf timeline home --limit 10 --page 2 # pagination

# Posts
frf post get <id>                     # post with full comments
frf post create <body>                # create post
frf post create <body> --group <name> # post to group
frf post update <id> <body>           # edit post
frf post delete <id>                  # delete post
frf post like <id>                    # like
frf post unlike <id>                  # unlike
frf post hide <id>                    # hide from feed
frf post unhide <id>                  # unhide

# Comments
frf comment add <post-id> <body>      # add comment
frf comment update <comment-id> <body># edit comment
frf comment delete <comment-id>       # delete comment

# Direct messages
frf direct create <body> --to user1,user2

# Search
frf search <query> --limit 10         # supports from:, intitle:, incomment:

# Users
frf user me                           # current user
frf user profile <username>           # user profile
frf user subscribers <username>       # followers
frf user subscriptions <username>     # following
frf user subscribe <username>         # follow
frf user unsubscribe <username>       # unfollow

# Groups
frf group list                        # my groups
frf group timeline <name> --limit 10  # group feed
```

Global flags: `--token <token>`, `--base-url <url>`

## Output

All commands produce plain text to stdout. No JSON mode — text is readable by both humans and agents.

Post format:
```
author (username)  2026-04-14T18:34:21Z
Post body text here
[3 likes, 5 comments]  id:uuid-here
```

Errors go to stderr as plain text. Exit code 1 on error.

## Configuration

Environment variables (loaded from `.env`):

| Variable | Required | Default | Description |
|---|---|---|---|
| `FREEFEED_BASE_URL` | no | `https://freefeed.net` | API base URL |
| `FREEFEED_USERNAME` | yes* | — | Username |
| `FREEFEED_PASSWORD` | yes* | — | Password |
| `FREEFEED_APP_TOKEN` | yes* | — | Auth token (alternative) |
| `FREEFEED_TIMELINE` | no | `home` | Default timeline (TUI) |
| `FREEFEED_TIMELINE_LIMIT` | no | `20` | Default limit (TUI) |

*Either token or username+password required.

## Building

```bash
go build -o bin/frf ./cmd/frf          # CLI
go build -o bin/frf-tui ./cmd/frf-tui  # TUI
```

## Code Conventions

- Go 1.22+, sync HTTP client
- CLI: cobra, text output via `fmt.Print*`, no JSON serialization layer
- Output helpers in `internal/cli/output.go` print directly from `client.*` types
- Client methods return typed Go structs, not raw JSON
- FreeFeed API returns inconsistent shapes (array vs map) — client handles this with multiple decode attempts

## API Client Methods

### Read
- `Authenticate()` — login with username/password
- `GetTimeline(type, username, limit, offset)` — any timeline
- `GetPost(id, maxComments)` — post with comments
- `SearchPosts(query, limit, offset)` — search
- `WhoAmI()` — current user profile
- `GetUserProfile(username)` — user profile
- `GetSubscribers(username)` / `GetSubscriptions(username)`
- `GetMyGroups()` — groups via whoami

### Write
- `CreatePost(body, feeds)` / `CreateDirectPost(body, recipients)`
- `UpdatePost(id, body)` / `DeletePost(id)`
- `LikePost(id)` / `UnlikePost(id)` / `HidePost(id)` / `UnhidePost(id)`
- `AddComment(postID, body)` / `UpdateComment(id, body)` / `DeleteComment(id)`
- `Subscribe(username)` / `Unsubscribe(username)`
