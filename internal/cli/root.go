package cli

import (
	"frf-tui/internal/app"
	"frf-tui/internal/client"

	"github.com/spf13/cobra"
)

var (
	flagBaseURL string
	flagToken   string
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "frf",
		Short:        "FreeFeed CLI",
		SilenceUsage: true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagBaseURL, "base-url", "", "FreeFeed API base URL (overrides env)")
	root.PersistentFlags().StringVar(&flagToken, "token", "", "auth token (overrides env)")

	root.AddCommand(newTimelineCmd())
	root.AddCommand(newPostCmd())
	root.AddCommand(newCommentCmd())
	root.AddCommand(newDirectCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newUserCmd())
	root.AddCommand(newGroupCmd())

	return root
}

func newClient() (*client.Client, error) {
	cfg, err := app.LoadConfig()
	if err != nil && flagToken == "" {
		return nil, err
	}

	baseURL := cfg.BaseURL
	if flagBaseURL != "" {
		baseURL = flagBaseURL
	}
	if baseURL == "" {
		baseURL = "https://freefeed.net"
	}

	token := flagToken
	if token == "" {
		token = cfg.Token
	}

	if token != "" {
		return client.NewWithToken(baseURL, token), nil
	}

	c := client.New(baseURL, cfg.Username, cfg.Password)
	if err := c.Authenticate(); err != nil {
		return nil, err
	}
	return c, nil
}
