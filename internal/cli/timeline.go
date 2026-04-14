package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	timelineLimit  int
	timelineOffset int
	timelinePage   int
)

func newTimelineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeline [type] [username]",
		Short: "Get a timeline feed",
		Long: `Get a FreeFeed timeline. Types:
  home         Home feed (default)
  discussions  Discussions
  directs      Direct messages
  posts        User's posts (requires username)
  likes        User's likes (requires username)
  comments     User's comments (requires username)`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			timelineType := "home"
			username := ""
			if len(args) >= 1 {
				timelineType = args[0]
			}
			if len(args) >= 2 {
				username = args[1]
			}

			switch timelineType {
			case "home", "discussions", "directs":
			case "posts", "likes", "comments":
				if username == "" {
					return fmt.Errorf("%s timeline requires a username argument", timelineType)
				}
			default:
				return fmt.Errorf("unknown timeline type: %s", timelineType)
			}

			if cmd.Flags().Changed("page") {
				timelineOffset = (timelinePage - 1) * timelineLimit
			}

			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			posts, err := c.GetTimeline(timelineType, username, timelineLimit, timelineOffset)
			if err != nil {
				return fmt.Errorf("fetch timeline: %w", err)
			}

			printPosts(posts)
			return nil
		},
	}

	cmd.Flags().IntVar(&timelineLimit, "limit", 20, "number of posts to fetch")
	cmd.Flags().IntVar(&timelineOffset, "offset", 0, "offset for pagination")
	cmd.Flags().IntVar(&timelinePage, "page", 1, "page number (1-based, overrides offset)")

	return cmd
}
