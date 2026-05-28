package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mreider/agilemarkdown/config"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// ConfigCommand exposes the project config as JSON for the VS Code
// extension's Settings tab. `show` emits the current state; `save`
// replaces the file with whatever JSON the caller pipes in on stdin.
// Round-trips through Defaults() so missing fields are filled.
var ConfigCommand = &cli.Command{
	Name:  "config",
	Usage: "Read or write .am/config.yaml as JSON",
	Commands: []*cli.Command{
		{
			Name:  "show",
			Usage: "Print the current config as JSON (resolved with defaults)",
			Action: func(ctx context.Context, c *cli.Command) error {
				root, err := findRootDirectory()
				if err != nil {
					return err
				}
				cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
				if err != nil {
					return err
				}
				return emitJSON(cfg)
			},
		},
		{
			Name:  "save",
			Usage: "Replace .am/config.yaml with JSON read from stdin",
			Action: func(ctx context.Context, c *cli.Command) error {
				root, err := findRootDirectory()
				if err != nil {
					return err
				}
				body, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				var cfg config.Config
				if err := json.Unmarshal(body, &cfg); err != nil {
					return fmt.Errorf("parse json: %w", err)
				}
				// Re-marshal as YAML and write. The Config.Save method
				// would also normalize but JSON → struct → YAML keeps
				// defaults and validation honest.
				yamlBytes, err := yaml.Marshal(&cfg)
				if err != nil {
					return fmt.Errorf("marshal yaml: %w", err)
				}
				path := filepath.Join(root, ".am", "config.yaml")
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(path, yamlBytes, 0644); err != nil {
					return err
				}
				fmt.Println("saved")
				return nil
			},
		},
	},
}
