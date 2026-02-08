package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"leetcli/internal/config"
	"leetcli/internal/store"
	"leetcli/internal/workspace"
)

var initProjectConfig bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize LeetCLI workspace and config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Config{
			Site: "https://leetcode.com",
			Workspace: config.WorkspaceConfig{
				ProblemsDir: "problems",
				DBPath:      filepath.Join(".leetcli", "leetcli.db"),
			},
		}

		path, err := config.Save(cfg, initProjectConfig)
		if err != nil {
			return err
		}

		if err := workspace.EnsureBaseDirs(cfg.Workspace.ProblemsDir); err != nil {
			return err
		}
		db, err := store.Open(cfg.Workspace.DBPath)
		if err != nil {
			return err
		}
		_ = db.Close()

		fmt.Printf("Initialized LeetCLI\n")
		fmt.Printf("Config: %s\n", path)
		fmt.Printf("DB: %s\n", cfg.Workspace.DBPath)
		fmt.Printf("Problems dir: %s\n", cfg.Workspace.ProblemsDir)
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Println("Hint: run git init to version your local practice workspace")
		}
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initProjectConfig, "project", false, "write project-local .leetcli/config.yaml instead of XDG config")
}
