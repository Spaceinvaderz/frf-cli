package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage comments",
	}

	cmd.AddCommand(newCommentAddCmd())
	cmd.AddCommand(newCommentUpdateCmd())
	cmd.AddCommand(newCommentDeleteCmd())

	return cmd
}

func newCommentAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <post-id> <body>",
		Short: "Add a comment to a post",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			comment, err := c.AddComment(args[0], args[1])
			if err != nil {
				return fmt.Errorf("add comment: %w", err)
			}

			fmt.Println("Comment added.")
			printComment(comment)
			return nil
		},
	}
}

func newCommentUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <comment-id> <body>",
		Short: "Update a comment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			if err := c.UpdateComment(args[0], args[1]); err != nil {
				return fmt.Errorf("update comment: %w", err)
			}

			fmt.Println("Comment updated.")
			return nil
		},
	}
}

func newCommentDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <comment-id>",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			if err := c.DeleteComment(args[0]); err != nil {
				return fmt.Errorf("delete comment: %w", err)
			}

			fmt.Println("Comment deleted.")
			return nil
		},
	}
}
