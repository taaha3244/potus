package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP servers",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List configured MCP servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "add [server-name]",
		Short: "Add MCP server interactively",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "remove [server-name]",
		Short: "Remove MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "test [server-name]",
		Short: "Test MCP server connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	})

	return cmd
}
