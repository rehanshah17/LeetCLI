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
var authCookieHeader string
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

		if strings.TrimSpace(authCookieHeader) != "" {
			s, c := extractAuthFromCookieHeader(authCookieHeader)
			if s != "" {
				cfg.Auth.LeetCodeSession = s
			}
			if c != "" {
				cfg.Auth.CSRFToken = c
			}
		}
		if strings.TrimSpace(authSession) != "" {
			cfg.Auth.LeetCodeSession = strings.TrimSpace(authSession)
		}
		if strings.TrimSpace(authCSRF) != "" {
			cfg.Auth.CSRFToken = strings.TrimSpace(authCSRF)
		}
		if cfg.Auth.LeetCodeSession == "" || cfg.Auth.CSRFToken == "" {
			return fmt.Errorf("missing credentials: use --cookie '<browser cookie header>' or --session/--csrf (or env LEETCODE_SESSION/CSRFTOKEN)")
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

var authGuideCmd = &cobra.Command{
	Use:   "guide",
	Short: "Show quickest way to obtain LeetCode cookies",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("1) Open https://leetcode.com in your browser and log in.")
		fmt.Println("2) Open DevTools -> Network and refresh the page.")
		fmt.Println("3) Click any request to leetcode.com and copy the full 'cookie' request header.")
		fmt.Println("4) Run: leet auth --cookie '<PASTED_COOKIE_HEADER>' --project")
		fmt.Println("   (or omit --project to save in XDG config).")
	},
}

func extractAuthFromCookieHeader(raw string) (session, csrf string) {
	parts := strings.Split(raw, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		switch strings.ToLower(k) {
		case "leetcode_session":
			session = v
		case "csrftoken":
			csrf = v
		}
	}
	return session, csrf
}

func init() {
	authCmd.Flags().StringVar(&authSession, "session", "", "LEETCODE_SESSION cookie")
	authCmd.Flags().StringVar(&authCSRF, "csrf", "", "csrftoken cookie")
	authCmd.Flags().StringVar(&authCookieHeader, "cookie", "", "full Cookie header from browser (auto-extracts LEETCODE_SESSION and csrftoken)")
	authCmd.Flags().BoolVar(&authProject, "project", false, "save to .leetcli/config.yaml")
	authCmd.AddCommand(authGuideCmd)
}
