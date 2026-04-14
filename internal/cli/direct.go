package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var directRecipients string

func newDirectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "direct",
		Short: "Direct messages",
	}

	cmd.AddCommand(newDirectCreateCmd())

	return cmd
}

func newDirectCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <body>",
		Short: "Send a direct message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if directRecipients == "" {
				return fmt.Errorf("--to is required (comma-separated usernames)")
			}

			recipients := strings.Split(directRecipients, ",")
			for i := range recipients {
				recipients[i] = strings.TrimSpace(recipients[i])
			}

			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			post, err := c.CreateDirectPost(args[0], recipients)
			if err != nil {
				return fmt.Errorf("send direct: %w", err)
			}

			fmt.Printf("Direct sent to %s.\n", directRecipients)
			printPost(post)
			return nil
		},
	}

	cmd.Flags().StringVar(&directRecipients, "to", "", "recipients (comma-separated usernames)")

	return cmd
}
