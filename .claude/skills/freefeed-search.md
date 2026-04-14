# Skill: freefeed-search

Search FreeFeed posts.

## When to use
When the user wants to find specific posts or content on FreeFeed.

## Commands

```bash
frf search "search terms"
frf search "from:username query"
frf search "intitle:keyword"
frf search "incomment:keyword"
frf search "from:alice intitle:travel" --limit 10 --page 2
```

## Operators
| Operator | Description |
|----------|-------------|
| `from:username` | Posts by user |
| `intitle:word` | In post body |
| `incomment:word` | In comments |
| `AND` / `OR` | Boolean |
