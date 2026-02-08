package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"leetcli/internal/config"
	"leetcli/internal/leetcode"
)

var authSession string
var authCSRF string
var authProject bool

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Configure LeetCode cookie authentication",
	RunE: func(cmd *cobra.Command, args []string) error {
		loaded, err := config.Load()
		if err != nil {
			return err
		}
		cfg := loaded.Config

		if strings.TrimSpace(authSession) != "" {
			cfg.Auth.LeetCodeSession = strings.TrimSpace(authSession)
		}
		if strings.TrimSpace(authCSRF) != "" {
			cfg.Auth.CSRFToken = strings.TrimSpace(authCSRF)
		}
		if cfg.Auth.LeetCodeSession == "" || cfg.Auth.CSRFToken == "" {
			return fmt.Errorf("missing credentials: set --session/--csrf or LEETCODE_SESSION/CSRFTOKEN")
		}

		cli := leetcode.New(cfg.Site, cfg.Auth.LeetCodeSession, cfg.Auth.CSRFToken)
		username, err := cli.ValidateAuth(context.Background())
		if err != nil {
			return err
		}

		path, err := config.Save(cfg, authProject)
		if err != nil {
			return err
		}
		fmt.Printf("Authenticated as %s\n", username)
		fmt.Printf("Saved auth config to %s\n", path)
		return nil
	},
}

func init() {
	authCmd.Flags().StringVar(&authSession, "session", "", "LEETCODE_SESSION cookie")
	authCmd.Flags().StringVar(&authCSRF, "csrf", "", "csrftoken cookie")
	authCmd.Flags().BoolVar(&authProject, "project", false, "save to .leetcli/config.yaml")
}
