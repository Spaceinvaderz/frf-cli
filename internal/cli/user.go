package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User profiles and social graph",
	}

	cmd.AddCommand(newUserMeCmd())
	cmd.AddCommand(newUserProfileCmd())
	cmd.AddCommand(newUserSubscribersCmd())
	cmd.AddCommand(newUserSubscriptionsCmd())
	cmd.AddCommand(newUserSubscribeCmd())
	cmd.AddCommand(newUserUnsubscribeCmd())

	return cmd
}

func newUserMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show current authenticated user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			profile, err := c.WhoAmI()
			if err != nil {
				return fmt.Errorf("whoami: %w", err)
			}

			printProfile(profile)
			return nil
		},
	}
}

func newUserProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile <username>",
		Short: "Show user profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			profile, err := c.GetUserProfile(args[0])
			if err != nil {
				return fmt.Errorf("profile: %w", err)
			}

			printProfile(profile)
			return nil
		},
	}
}

func newUserSubscribersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "subscribers <username>",
		Short: "List user's subscribers (followers)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			users, err := c.GetSubscribers(args[0])
			if err != nil {
				return fmt.Errorf("subscribers: %w", err)
			}

			printUserList("Subscribers", users)
			return nil
		},
	}
}

func newUserSubscriptionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "subscriptions <username>",
		Short: "List user's subscriptions (following)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			users, err := c.GetSubscriptions(args[0])
			if err != nil {
				return fmt.Errorf("subscriptions: %w", err)
			}

			printUserList("Subscriptions", users)
			return nil
		},
	}
}

func newUserSubscribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "subscribe <username>",
		Short: "Subscribe to a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			if err := c.Subscribe(args[0]); err != nil {
				return fmt.Errorf("subscribe: %w", err)
			}

			fmt.Printf("Subscribed to %s.\n", args[0])
			return nil
		},
	}
}

func newUserUnsubscribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unsubscribe <username>",
		Short: "Unsubscribe from a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			if err := c.Unsubscribe(args[0]); err != nil {
				return fmt.Errorf("unsubscribe: %w", err)
			}

			fmt.Printf("Unsubscribed from %s.\n", args[0])
			return nil
		},
	}
}
