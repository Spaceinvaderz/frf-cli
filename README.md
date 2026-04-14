# frf

FreeFeed CLI & TUI client in Go.

## Install

```bash
go install github.com/Spaceinvaderz/frf-cli/cmd/frf@latest
```

Or build from source:

```bash
git clone git@github.com:Spaceinvaderz/frf-cli.git
cd frf-cli
go build -o bin/frf ./cmd/frf
go build -o bin/frf-tui ./cmd/frf-tui
```

## Auth

Create `.env` in the project root:

```env
FREEFEED_APP_TOKEN=your_token_here
```

Or use username/password:

```env
FREEFEED_USERNAME=your_username
FREEFEED_PASSWORD=your_password
```

Token can also be passed via `--token` flag.

## Usage

```bash
# Read
frf timeline                          # home feed
frf timeline discussions              # discussions
frf timeline directs                  # direct messages
frf timeline posts <username>         # user's posts
frf post get <id>                     # post with comments
frf search "query"                    # search (from:, intitle:, incomment:)

# Write
frf post create "Hello, FreeFeed!"
frf post create "Post" --group mygroup
frf comment add <post-id> "Nice post!"
frf direct create "Hey" --to user1,user2

# Social
frf user me
frf user profile <username>
frf user subscribers <username>
frf user subscribe <username>
frf group list
frf group timeline <name>

# Post actions
frf post like <id>
frf post unlike <id>
frf post update <id> "new text"
frf post delete <id>
```

Pagination: `--limit 10 --page 2`

## TUI

```bash
frf-tui
```

Two-panel terminal interface: post list + detail view with comments.

Keys: `h` Home, `m` Direct, `D` Discussions, `Tab` switch panes, `j/k` scroll, `a` all comments, `q` quit.

## Project structure

```
cmd/frf/          CLI (cobra)
cmd/frf-tui/      TUI (bubbletea)
internal/client/  FreeFeed API client (shared)
internal/cli/     CLI commands and output
internal/app/     Config (shared) + TUI model
```

## License

See [LICENSE](LICENSE).
