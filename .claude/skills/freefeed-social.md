# Skill: freefeed-social

Manage FreeFeed social connections.

## When to use
When the user wants to check profiles, manage subscriptions, or browse groups.

## Commands

```bash
frf user me
frf user profile <username>
frf user subscribers <username>
frf user subscriptions <username>
frf user subscribe <username>
frf user unsubscribe <username>
frf group list
frf group timeline <groupname> --limit 10
```

## Notes
- Follow/unfollow are immediate — confirm with user first
- `frf group list` shows all groups (can be 100+)
