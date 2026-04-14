package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	searchLimit  int
	searchOffset int
	searchPage   int
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search posts",
		Long: `Search FreeFeed posts. Supports operators:
  from:username    Posts by user
  intitle:word     Search in post body
  incomment:word   Search in comments
  AND / OR         Boolean operators`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("page") {
				searchOffset = (searchPage - 1) * searchLimit
			}

			c, err := newClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			posts, err := c.SearchPosts(args[0], searchLimit, searchOffset)
			if err != nil {
				return fmt.Errorf("search: %w", err)
			}

			printPosts(posts)
			return nil
		},
	}

	cmd.Flags().IntVar(&searchLimit, "limit", 20, "number of results")
	cmd.Flags().IntVar(&searchOffset, "offset", 0, "offset for pagination")
	cmd.Flags().IntVar(&searchPage, "page", 1, "page number")

	return cmd
}
