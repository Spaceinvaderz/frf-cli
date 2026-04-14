package cli

import (
	"fmt"
	"strings"
	"time"

	"frf-tui/internal/client"
)

func printPosts(posts []client.Post) {
	for i, p := range posts {
		if i > 0 {
			fmt.Println(strings.Repeat("─", 60))
		}
		printPost(p)
	}
}

func printPost(p client.Post) {
	fmt.Printf("%s  %s\n", formatAuthor(p.User), formatTimestamp(p.CreatedAt))
	fmt.Println(p.Body)
	var meta []string
	if p.LikesCount > 0 {
		meta = append(meta, fmt.Sprintf("%d likes", p.LikesCount))
	}
	if p.CommentsCount > 0 {
		meta = append(meta, fmt.Sprintf("%d comments", p.CommentsCount))
	}
	if len(meta) > 0 {
		fmt.Printf("[%s]  id:%s\n", strings.Join(meta, ", "), p.ID)
	} else {
		fmt.Printf("id:%s\n", p.ID)
	}
}

func printPostFull(p client.Post) {
	printPost(p)
	if len(p.Comments) > 0 {
		fmt.Println()
		fmt.Printf("── Comments (%d) ──\n", p.CommentsCount)
		for _, c := range p.Comments {
			fmt.Printf("\n  %s  %s\n", formatAuthor(c.User), formatTimestamp(c.CreatedAt))
			fmt.Printf("  %s\n", c.Body)
		}
	}
}

func printComment(c client.Comment) {
	fmt.Printf("  %s  %s\n", formatAuthor(c.User), formatTimestamp(c.CreatedAt))
	fmt.Printf("  %s\n", c.Body)
}

func printProfile(p client.UserProfile) {
	fmt.Println(formatProfileName(p))
	if p.Type != "" {
		fmt.Printf("type: %s\n", p.Type)
	}
	if p.Description != "" {
		fmt.Println(p.Description)
	}
	fmt.Printf("subscribers: %d  subscriptions: %d\n", p.SubscriberCount, p.SubscriptionCount)
	if p.IsPrivate == "1" {
		fmt.Println("[private]")
	}
	fmt.Printf("id: %s\n", p.ID)
}

func printUserList(label string, users []client.User) {
	fmt.Printf("%s (%d)\n", label, len(users))
	for _, u := range users {
		fmt.Printf("  %s\n", formatAuthor(u))
	}
}

func printGroupList(groups []client.User) {
	fmt.Printf("Groups (%d)\n", len(groups))
	for _, g := range groups {
		fmt.Printf("  %s\n", formatAuthor(g))
	}
}

func formatAuthor(u client.User) string {
	if u.ScreenName != "" && u.Username != "" && u.ScreenName != u.Username {
		return fmt.Sprintf("%s (%s)", u.ScreenName, u.Username)
	}
	if u.ScreenName != "" {
		return u.ScreenName
	}
	if u.Username != "" {
		return u.Username
	}
	return "unknown"
}

func formatProfileName(p client.UserProfile) string {
	if p.ScreenName != "" && p.ScreenName != p.Username {
		return fmt.Sprintf("%s (%s)", p.ScreenName, p.Username)
	}
	return p.Username
}

func formatTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	ms, err := parseInt64(raw)
	if err != nil {
		return raw
	}
	var t time.Time
	if ms > 1_000_000_000_000 {
		t = time.UnixMilli(ms)
	} else {
		t = time.Unix(ms, 0)
	}
	return t.UTC().Format(time.RFC3339)
}

func parseInt64(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}
