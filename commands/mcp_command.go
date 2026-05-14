package commands

import (
	"context"

	"github.com/mreider/agilemarkdown/mcpserver"
	"github.com/urfave/cli/v3"
)

// NewMCPCommand builds the `am mcp` subcommand.
func NewMCPCommand(version string) *cli.Command {
	return &cli.Command{
		Name:      "mcp",
		Usage:     "Run an MCP (Model Context Protocol) stdio server exposing this backlog",
		ArgsUsage: " ",
		Action: func(ctx context.Context, c *cli.Command) error {
			rootDir, err := findRootDirectory()
			if err != nil {
				rootDir = "."
			}
			return mcpserver.Run(ctx, rootDir, version)
		},
	}
}
