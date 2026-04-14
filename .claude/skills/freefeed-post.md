# Skill: freefeed-post

Create, edit, and interact with FreeFeed posts and comments.

## When to use
When the user wants to create posts, reply to discussions, or manage content. ALWAYS confirm with the user before posting.

## Commands

```bash
frf post create "Post body text"
frf post create "Post body" --group groupname
frf post update <post-id> "Updated body"
frf post delete <post-id>
frf post like <post-id>
frf post unlike <post-id>
frf post hide <post-id>
frf post unhide <post-id>
frf comment add <post-id> "Comment text"
frf comment update <comment-id> "Updated text"
frf comment delete <comment-id>
frf direct create "Message" --to user1,user2
```

## Important
- ALWAYS show the user what will be posted before executing
- NEVER auto-post without explicit confirmation
- Can't like own posts (API returns 403)
