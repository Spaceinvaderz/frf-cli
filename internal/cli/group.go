package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	groupTimelineLimit  int
	groupTimelineOffset int
	groupTimelinePage   int
)

func newGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Browse groups",
	}

	cmd.AddCommand(newGroupListCmd())
	cmd.AddCommand(newGroupTimelineCmd())

	return cmd
}

func newGroupListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your groups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			groups, err := c.GetMyGroups()
			if err != nil {
				return fmt.Errorf("groups: %w", err)
			}

			printGroupList(groups)
			return nil
		},
	}
}

func newGroupTimelineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeline <group-name>",
		Short: "Get group feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("page") {
				groupTimelineOffset = (groupTimelinePage - 1) * groupTimelineLimit
			}

			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			posts, err := c.GetTimeline("posts", args[0], groupTimelineLimit, groupTimelineOffset)
			if err != nil {
				return fmt.Errorf("group timeline: %w", err)
			}

			printPosts(posts)
			return nil
		},
	}

	cmd.Flags().IntVar(&groupTimelineLimit, "limit", 20, "number of posts")
	cmd.Flags().IntVar(&groupTimelineOffset, "offset", 0, "offset for pagination")
	cmd.Flags().IntVar(&groupTimelinePage, "page", 1, "page number")

	return cmd
}
