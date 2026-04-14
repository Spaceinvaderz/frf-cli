# Skill: freefeed-read

Read FreeFeed timelines, posts, and comments using the `frf` CLI.

## When to use
When the user wants to read their FreeFeed feed, check a specific post, or browse comments.

## Commands

```bash
frf timeline                        # home feed
frf timeline discussions            # discussions
frf timeline directs --limit 10     # direct messages
frf timeline posts <username>       # user's posts
frf post get <id>                   # post with all comments
frf user profile <username>         # user profile
frf user me                         # current user
frf user subscribers <username>     # followers
frf user subscriptions <username>   # following
frf group list                      # my groups
frf group timeline <name> --limit 5 # group feed
frf search "query" --limit 10       # search (from:, intitle:, incomment:)
```

## Pagination
```bash
frf timeline --limit 10 --page 2
frf search "query" --page 3
```

## Notes
- All output is plain text
- Post IDs are UUIDs
- Auth via .env (FREEFEED_APP_TOKEN or FREEFEED_USERNAME + FREEFEED_PASSWORD)
