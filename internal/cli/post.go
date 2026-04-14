package cli

import (
	"fmt"

	"frf-tui/internal/client"

	"github.com/spf13/cobra"
)

var postCreateGroup string

func newPostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post",
		Short: "Manage posts",
	}

	cmd.AddCommand(newPostGetCmd())
	cmd.AddCommand(newPostCreateCmd())
	cmd.AddCommand(newPostUpdateCmd())
	cmd.AddCommand(newPostDeleteCmd())
	cmd.AddCommand(newPostLikeCmd())
	cmd.AddCommand(newPostUnlikeCmd())
	cmd.AddCommand(newPostHideCmd())
	cmd.AddCommand(newPostUnhideCmd())

	return cmd
}

func newPostGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <post-id>",
		Short: "Get a post with comments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			post, err := c.GetPost(args[0], "all")
			if err != nil {
				return fmt.Errorf("fetch post: %w", err)
			}

			printPostFull(post)
			return nil
		},
	}
}

func newPostCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <body>",
		Short: "Create a new post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			var feeds []string
			if postCreateGroup != "" {
				feeds = []string{postCreateGroup}
			}

			post, err := c.CreatePost(args[0], feeds)
			if err != nil {
				return fmt.Errorf("create post: %w", err)
			}

			fmt.Println("Post created.")
			printPost(post)
			return nil
		},
	}

	cmd.Flags().StringVar(&postCreateGroup, "group", "", "post to a group")

	return cmd
}

func newPostUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <post-id> <body>",
		Short: "Update a post",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			if err := c.UpdatePost(args[0], args[1]); err != nil {
				return fmt.Errorf("update post: %w", err)
			}

			fmt.Println("Post updated.")
			return nil
		},
	}
}

func newPostDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <post-id>",
		Short: "Delete a post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			if err := c.DeletePost(args[0]); err != nil {
				return fmt.Errorf("delete post: %w", err)
			}

			fmt.Println("Post deleted.")
			return nil
		},
	}
}

func newPostLikeCmd() *cobra.Command {
	return newPostActionCmd("like", "Like a post", (*client.Client).LikePost)
}

func newPostUnlikeCmd() *cobra.Command {
	return newPostActionCmd("unlike", "Remove like from a post", (*client.Client).UnlikePost)
}

func newPostHideCmd() *cobra.Command {
	return newPostActionCmd("hide", "Hide a post from feed", (*client.Client).HidePost)
}

func newPostUnhideCmd() *cobra.Command {
	return newPostActionCmd("unhide", "Unhide a post", (*client.Client).UnhidePost)
}

func newPostActionCmd(name, short string, action func(*client.Client, string) error) *cobra.Command {
	return &cobra.Command{
		Use:   name + " <post-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			if err := action(c, args[0]); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}

			fmt.Println("Done.")
			return nil
		},
	}
}
